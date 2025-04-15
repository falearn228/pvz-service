package queries

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"pvz-service/internal/db"
	"pvz-service/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

// ProductQueries содержит методы запросов для работы с товарами
type ProductQueries struct {
	db *db.Database
	sq squirrel.StatementBuilderType
}

// NewProductQueries создает новый экземпляр ProductQueries
func NewProductQueries(db *db.Database) *ProductQueries {
	return &ProductQueries{
		db: db,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db),
	}
}

// AddProduct добавляет товар в приёмку
func (q *ProductQueries) AddProduct(ctx context.Context, receptionID, productType string) (*models.Product, error) {
	// Генерируем UUID
	id := uuid.New().String()
	now := time.Now()

	// Создаем запрос
	query := q.sq.
		Insert("product").
		Columns("id", "datetime", "type", "reception_id").
		Values(id, now, productType, receptionID).
		Suffix("RETURNING id, datetime, type, reception_id")

	qsql, args, err := query.ToSql()
	log.Printf("SQL: %s, Args: %v", qsql, args)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var product models.Product
	err = q.db.QueryRowxContext(ctx, qsql, args...).StructScan(&product)
	if err != nil {
		return nil, fmt.Errorf("failed to add product: %w", err)
	}

	return &product, nil
}

// GetLastProductFromReception получает последний добавленный товар в приёмку
func (q *ProductQueries) GetLastProductFromReception(ctx context.Context, receptionID string) (*models.Product, error) {
	query := q.sq.
		Select("id", "datetime", "type", "reception_id").
		From("product").
		Where(squirrel.Eq{"reception_id": receptionID}).
		OrderBy("datetime DESC").
		Limit(1)

	qsql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var product models.Product
	err = q.db.QueryRowxContext(ctx, qsql, args...).StructScan(&product)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no products found in reception %s", receptionID)
		}
		return nil, fmt.Errorf("failed to get last product: %w", err)
	}

	return &product, nil
}

// DeleteProduct удаляет товар по ID
func (q *ProductQueries) DeleteProduct(ctx context.Context, productID string) error {
	query := q.sq.
		Delete("product").
		Where(squirrel.Eq{"id": productID})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	result, err := q.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product with id %s not found", productID)
	}

	return nil
}

// GetProductsByReception получает все товары для приёмки
func (q *ProductQueries) GetProductsByReception(ctx context.Context, receptionID string) ([]models.Product, error) {
	query := q.sq.
		Select("id", "datetime", "type", "reception_id").
		From("product").
		Where(squirrel.Eq{"reception_id": receptionID}).
		OrderBy("datetime DESC")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var products []models.Product
	err = q.db.SelectContext(ctx, &products, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get products: %w", err)
	}

	return products, nil
}
