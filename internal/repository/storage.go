package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/k057ya/gophermart-loyalty/internal/model"
	"go.uber.org/zap"
)

type DBStorage struct {
	connection *sql.DB
	log        *zap.SugaredLogger
}

func (storage *DBStorage) Close() error {
	return storage.connection.Close()
}

func (storage *DBStorage) Migrate(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (storage *DBStorage) GetOrdersToProcess(ctx context.Context) []model.Order {
	var orders []model.Order
	// TODO fill orders from db
	return orders
}

func (storage *DBStorage) UpdateOrders(ctx context.Context, batch []model.Order) error {
	// TODO
	return nil
}

func NewDBStorage(dsn string, log *zap.SugaredLogger) (*DBStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	var storage = &DBStorage{
		connection: db,
		log:        log,
	}
	err = storage.Migrate(db)
	return storage, err
}
