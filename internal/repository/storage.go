package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/k057ya/gophermart-loyalty/internal/model"
	"go.uber.org/zap"
)

var ErrLoginTaken = errors.New("login is taken")

type DBStorage struct {
	connection *sql.DB
	log        *zap.SugaredLogger
}

func (s *DBStorage) Close() error {
	return s.connection.Close()
}

func (s *DBStorage) Migrate(db *sql.DB) error {
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
//
// ORDERS
//
func (s *DBStorage) OrdersGetForProcessing(ctx context.Context) []model.Order {
	var orders []model.Order
	// TODO fill orders from db
	return orders
}

func (storage *DBStorage) OrdersUpdate(ctx context.Context, batch []model.Order) error {
	// TODO
	return nil
}

//
// USERS
//
func (s *DBStorage) UserCreate(ctx context.Context, user *model.User) error {
	var id int64
	err := s.connection.QueryRowContext(
		ctx,
		`INSERT INTO users (login, password, created_at)
		 VALUES ($1, $2, NOW())
		 ON CONFLICT (login) DO NOTHING
		 RETURNING id`,
		user.Login, user.Password,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrLoginTaken
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	user.ID = id
	return nil
}

func (s *DBStorage) UserGetByLogin(login string) (model.User, error) {
	var user model.User

	err := s.connection.QueryRow(
		`SELECT id, login, password, created_at
		 FROM users
		 WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &user.Password, &user.CreatedAt)
	if err != nil {
		return user, fmt.Errorf("get user by login: %w", err)
	}
	return user, nil
}

func (s *DBStorage) UserCreateToken(ctx context.Context, user *model.User) (string, error) {
	var randomBytes [32]byte
	rand.Read(randomBytes[:])
	token := base64.RawURLEncoding.EncodeToString(randomBytes[:])
	hash := sha256.Sum256([]byte(token))

	_, err := s.connection.ExecContext(
		ctx,
		`INSERT INTO tokens (user_id, token_hash, valid_until)
		 VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '24 hours')`,
		user.ID, hash[:],
	)
	if err != nil {
		return "", fmt.Errorf("create user token: %w", err)
	}
	return token, nil
}

func (s *DBStorage) UserGetByToken(ctx context.Context, token string) (model.User, error) {
	hash := sha256.Sum256([]byte(token))
	var user model.User
	var createdAt sql.NullTime
	err := s.connection.QueryRowContext(
		ctx,
		`SELECT u.id, u.login, u.created_at
		 FROM tokens AS t
		 JOIN users AS u ON u.id = t.user_id
		 WHERE t.token_hash = $1 AND t.valid_until > CURRENT_TIMESTAMP`,
		hash[:],
	).Scan(&user.ID, &user.Login, &createdAt)
	if err != nil {
		return model.User{}, fmt.Errorf("get user by token: %w", err)
	}

	user.CreatedAt = createdAt.Time
	return user, nil
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
	var s = &DBStorage{
		connection: db,
		log:        log,
	}
	err = s.Migrate(db)
	return s, err
}
