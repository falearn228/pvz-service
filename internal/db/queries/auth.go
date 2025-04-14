package queries

import (
	"context"
	sqlPackage "database/sql"
	"fmt"
	"pvz-service/internal/db"
	"pvz-service/internal/models"

	"github.com/Masterminds/squirrel"
)

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
func (q *AuthQueries) CreateUser(ctx context.Context, user *models.User) (string, error) {
	query := q.sq.
		Insert("users").
		Columns("email", "role", "password").
		Values(user.Email, user.Role, user.Password).
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

	sql, args, err := query.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to build query: %w", err)
	}

	var exists int
	err = q.db.QueryRowContext(ctx, sql, args...).Scan(&exists)
	if err != nil {
		if err == sqlPackage.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return true, nil
}
