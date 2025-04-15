package queries

import (
	"context"
	"database/sql"
	"fmt"
	"pvz-service/internal/db"
	"pvz-service/internal/models"

	"github.com/Masterminds/squirrel"
)

// AuthQueriesInterface определяет интерфейс для запросов, связанных с аутентификацией
type AuthQueriesInterface interface {
	GetUserByEmail(ctx context.Context, email string) (bool, error)
	CreateUser(ctx context.Context, email, passwordHash, role string) (string, error)
	GetUserWithCredentials(ctx context.Context, email string) (*models.User, error)
}

// AuthQueries содержит методы запросов для авторизации
type AuthQueries struct {
	db *db.Database
	sq squirrel.StatementBuilderType
}

// NewAuthQueries создает новый экземпляр AuthQueries
func NewAuthQueries(db *db.Database) *AuthQueries {
	return &AuthQueries{
		db: db,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db),
	}
}

// CreateUser создает нового пользователя
func (q *AuthQueries) CreateUser(ctx context.Context, email, passwordHash, role string) (string, error) {
	query := q.sq.
		Insert("users").
		Columns("email", "password_hash", "role", "created_at").
		Values(email, passwordHash, role, squirrel.Expr("CURRENT_TIMESTAMP")).
		Suffix("RETURNING id")

	sql, args, err := query.ToSql()
	if err != nil {
		return "", fmt.Errorf("failed to build query: %w", err)
	}

	var id string
	err = q.db.QueryRowContext(ctx, sql, args...).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}

	return id, nil
}

// GetUserByEmail проверяет, существует ли пользователь с таким email
func (q *AuthQueries) GetUserByEmail(ctx context.Context, email string) (bool, error) {
	query := q.sq.
		Select("1").
		From("users").
		Where(squirrel.Eq{"email": email}).
		Limit(1)

	qsql, args, err := query.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build query: %w", err)
	}

	var exists int
	err = q.db.QueryRowContext(ctx, qsql, args...).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return true, nil
}

// GetUserWithCredentials получает пользователя по email вместе с хешем пароля
func (q *AuthQueries) GetUserWithCredentials(ctx context.Context, email string) (*models.User, error) {
	query := q.sq.
		Select("id", "email", "role", "password_hash").
		From("users").
		Where(squirrel.Eq{"email": email}).
		Limit(1)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var user models.User
	err = q.db.GetContext(ctx, &user, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
