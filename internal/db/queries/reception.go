package queries

import (
	"context"
	"fmt"
	"time"

	"database/sql"
	"pvz-service/internal/db"
	"pvz-service/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

// ReceptionQueries содержит методы запросов для работы с приёмками
type ReceptionQueries struct {
	db *db.Database
	sq squirrel.StatementBuilderType
}

// NewReceptionQueries создает новый экземпляр ReceptionQueries
func NewReceptionQueries(db *db.Database) *ReceptionQueries {
	return &ReceptionQueries{
		db: db,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db),
	}
}

// CheckOpenReception проверяет, есть ли уже открытая приёмка для данного ПВЗ
func (q *ReceptionQueries) CheckOpenReception(ctx context.Context, pvzID string) (bool, error) {
	query := q.sq.
		Select("1").
		From("reception").
		Where(squirrel.Eq{"pvz_id": pvzID, "status": "in_progress"}).
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
		return false, fmt.Errorf("failed to check open reception: %w", err)
	}

	return true, nil
}

// CreateReception создает новую приёмку товаров
func (q *ReceptionQueries) CreateReception(ctx context.Context, pvzID string) (*models.Reception, error) {
	// Генерируем UUID
	id := uuid.New().String()
	now := time.Now()

	// Создаем запрос
	query := q.sq.
		Insert("reception").
		Columns("id", "datetime", "pvz_id", "status").
		Values(id, now, pvzID, "in_progress").
		Suffix("RETURNING id, datetime, pvz_id, status")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var reception models.Reception
	err = q.db.QueryRowxContext(ctx, sql, args...).StructScan(&reception)
	if err != nil {
		return nil, fmt.Errorf("failed to create reception: %w", err)
	}

	return &reception, nil
}

// GetLastOpenReception получает последнюю открытую приёмку для ПВЗ
func (q *ReceptionQueries) GetLastOpenReception(ctx context.Context, pvzID string) (*models.Reception, error) {
	query := q.sq.
		Select("id", "datetime", "pvz_id", "status").
		From("reception").
		Where(squirrel.Eq{"pvz_id": pvzID, "status": "in_progress"}).
		OrderBy("datetime DESC").
		Limit(1)

	qsql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var reception models.Reception
	err = q.db.QueryRowxContext(ctx, qsql, args...).StructScan(&reception)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no open reception found for pvz %s", pvzID)
		}
		return nil, fmt.Errorf("failed to get open reception: %w", err)
	}

	return &reception, nil
}

// CloseReception закрывает приёмку товаров
func (q *ReceptionQueries) CloseReception(ctx context.Context, receptionID string) (*models.Reception, error) {
	query := q.sq.
		Update("reception").
		Set("status", "close").
		Where(squirrel.Eq{"id": receptionID}).
		Suffix("RETURNING id, datetime, pvz_id, status")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var reception models.Reception
	err = q.db.QueryRowxContext(ctx, sql, args...).StructScan(&reception)
	if err != nil {
		return nil, fmt.Errorf("failed to close reception: %w", err)
	}

	return &reception, nil
}

// GetReceptionsByPVZ получает все приёмки для ПВЗ
func (q *ReceptionQueries) GetReceptionsByPVZ(ctx context.Context, pvzID string) ([]models.Reception, error) {
	query := q.sq.
		Select("id", "datetime", "pvz_id", "status").
		From("reception").
		Where(squirrel.Eq{"pvz_id": pvzID}).
		OrderBy("datetime DESC")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var receptions []models.Reception
	err = q.db.SelectContext(ctx, &receptions, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receptions: %w", err)
	}

	return receptions, nil
}
