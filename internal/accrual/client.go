package accrual

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/k057ya/gophermart-loyalty/internal/model"
	"github.com/k057ya/gophermart-loyalty/internal/repository"
	"go.uber.org/zap"
)

type Client struct {
	Client      HTTPClient
	Storage     *repository.DBStorage
	Log         *zap.SugaredLogger
	WorkerCount int
}

type HTTPClient struct {
	*resty.Client
}

type retryPause struct {
	mu       sync.Mutex
	resumeAt time.Time
}

func (p *retryPause) delay(d time.Duration) {
	until := time.Now().Add(d)

	p.mu.Lock()
	defer p.mu.Unlock()

	if until.After(p.resumeAt) {
		p.resumeAt = until
	}
}

func (p *retryPause) wait(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		p.mu.Lock()
		remaining := time.Until(p.resumeAt)
		p.mu.Unlock()

		if remaining <= 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(remaining):
		}
	}
}

func NewHTTPClient(baseURL string) HTTPClient {
	hc := HTTPClient{resty.New().
		SetTimeout(10*time.Second).
		SetHeader("Content-Type", "application/json").
		SetBaseURL(baseURL)}

	return hc
}

func (hc *HTTPClient) RequestContext(ctx context.Context, url string) (*resty.Response, error) {
	return hc.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		Get(url)
}

func NewClient(hc HTTPClient, storage *repository.DBStorage, log *zap.SugaredLogger) *Client {
	return &Client{
		Client:      hc,
		Storage:     storage,
		Log:         log,
		WorkerCount: 3,
	}
}

func Run(ctx context.Context, c *Client) error {

	var wg sync.WaitGroup
	// отдельный контекст для отправки
	sendCtx, cancelSend := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelSend()

	//
	// КОЛЛЕКТОР
	//
	// канал для заказов из БД
	ochan := make(chan model.Order)
	wg.Go(func() { c.collector(ctx, ochan) })

	//
	// СИНКЕР
	//
	// канал с обновленными результатами из внешней системы
	reschan := make(chan model.Order)
	// хранить паузу для too-many-requests
	var pause retryPause
	// отдельная wg для воркеров
	var wrkWg sync.WaitGroup
	for id := range c.WorkerCount {
		wrkWg.Go(func() { c.syncer(ctx, sendCtx, ochan, reschan, &pause, c.Log.Named("client-worker-"+strconv.Itoa(id))) })
	}

	//
	// БАТЧЕР
	//
	// канал для батчей обработанных заказов
	bchan := make(chan []model.Order)
	wg.Go(func() { c.batcher(sendCtx, reschan, bchan) })
	wg.Go(func() {
		wrkWg.Wait()
		close(reschan)
	})

	//
	// АПДЕЙТЕР
	//
	wg.Go(func() { c.updater(sendCtx, bchan) })

	//
	// Обработка Graceful shutdown
	//
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	<-ctx.Done()

	// пробуем доотправить накопленное
	shutdownTimeout := 5 * time.Second
	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()

	select {
	case <-done:
		return nil

	// по истечению таймаута убиваем все запросы
	// и ждем завершения воркеров
	// и возвращаем ошибку
	case <-timer.C:
		cancelSend()

		<-done

		return fmt.Errorf(
			"workers were not finished in timeout %s: %w",
			shutdownTimeout,
			context.DeadlineExceeded,
		)
	}
}

// собирает заказы из Бд до отмены или ошибки контекста
// с указанной периодичностью
func (c Client) collector(ctx context.Context, ochan chan model.Order) {

	pollInterval := 1 * time.Second
	defer close(ochan)

	for ctx.Err() == nil {
		for _, order := range c.Storage.GetOrdersToProcess(ctx) {
			select {
			case <-ctx.Done():
				return
			case ochan <- order:
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
		}
	}
}

// ждет заказы, которые надо обработать
// для каждого стучит во внешнюю систему
// создает обновленную модель и отправляет в канал с результатами
//
// если внешняя система рапортует Too many requests, тормозит
// процесс забора информации для всех воркеров на указанное в
// Retry-after время

func (c Client) syncer(
	ctx context.Context,
	sendCtx context.Context,
	ochan <-chan model.Order,
	reschan chan<- model.Order,
	pause *retryPause,
	log *zap.SugaredLogger) {

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		// нужно ли ждать сервис
		pause.wait(ctx)

		select {
		// контекст закрыт - сворачиваемся
		case <-ctx.Done():
			return
		case order, ok := <-ochan:
			// канал с заказами закрыт - сворачиваемся
			if !ok {
				return
			}

			// повторно проверяем resumeAt
			// и если запрос требует ожидания
			// подвешиваем выполнение fetch
		loop:
			for {
				if ctx.Err() != nil {
					return
				}

				// проверить нужно ли ждать сервис
				pause.wait(ctx)

				updated, retryAfter, err := c.fetch(ctx, order)
				// ошибка
				if err != nil {
					log.Warnw("error fetching order", err)
					break
				}
				// too many reqs, подождать
				if retryAfter != nil {
					pause.delay(*retryAfter)
					continue loop
				}

				select {
				case <-sendCtx.Done():
					return
				case reschan <- updated:
					break loop
				}

			}
		}
	}

}

// batcher ждет обноленные после синка результаты, добавляет в батч
// батч пушится по таймеру или отмене контекста
func (c Client) batcher(ctx context.Context, reschan <-chan model.Order, bchan chan<- []model.Order) {
	defer close(bchan)
	batchTicker := time.NewTicker(5 * time.Second)
	defer batchTicker.Stop()

	var batch []model.Order

	// отправить накопленные результаты
	var push = func() {
		if len(batch) != 0 {
			select {
			case <-ctx.Done():
				return
			case bchan <- batch:
				batch = nil
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-batchTicker.C:
			push()
		case order, ok := <-reschan:
			if !ok {
				push()
				return
			}
			batch = append(batch, order)
		}
	}
}

// получает в канале готовые для апдейта батчи
// и обновляет в БД
func (c Client) updater(ctx context.Context, bchan <-chan []model.Order) {
	for {
		select {
		case <-ctx.Done():
			return
		case batch, ok := <-bchan:
			if !ok {
				return
			}
			err := c.Storage.UpdateOrders(ctx, batch)
			if err != nil {
				c.Log.Warn("error updating orders", err)
			}
		}
	}
}

func (c Client) fetch(ctx context.Context, order model.Order) (model.Order, *time.Duration, error) {
	// make request
	retry := 0 * time.Second
	return model.Order{}, &retry, nil
}
