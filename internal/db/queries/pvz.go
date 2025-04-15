// internal/db/queries/pvz.go
package queries

import (
	"context"
	"fmt"
	"time"

	"pvz-service/internal/db"
	"pvz-service/internal/models"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

// PVZQueriesInterface определяет интерфейс для запросов к ПВЗ
type PVZQueriesInterface interface {
	CreatePVZ(ctx context.Context, city string) (*models.PVZ, error)
	GetPVZList(ctx context.Context, params models.PVZListQuery) ([]models.PVZ, int, error)
}

// PVZQueries содержит методы запросов для работы с ПВЗ
type PVZQueries struct {
	db *db.Database
	sq squirrel.StatementBuilderType
}

// NewPVZQueries создает новый экземпляр PVZQueries
func NewPVZQueries(db *db.Database) *PVZQueries {
	return &PVZQueries{
		db: db,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db),
	}
}

// CreatePVZ создает новый ПВЗ
func (q *PVZQueries) CreatePVZ(ctx context.Context, city string) (*models.PVZ, error) {
	// Генерируем UUID
	id := uuid.New().String()
	now := time.Now()

	// Создаем запрос
	query := q.sq.
		Insert("pvz").
		Columns("id", "city", "registration_date").
		Values(id, city, now).
		Suffix("RETURNING id, city, registration_date")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var pvz models.PVZ
	err = q.db.QueryRowxContext(ctx, sql, args...).StructScan(&pvz)
	if err != nil {
		return nil, fmt.Errorf("failed to create pvz: %w", err)
	}

	return &pvz, nil
}

// GetPVZList получает список ПВЗ с фильтрацией и пагинацией
func (q *PVZQueries) GetPVZList(ctx context.Context, params models.PVZListQuery) ([]models.PVZ, int, error) {
	// Формируем базовый запрос
	queryBuilder := q.sq.
		Select("id", "registration_date", "city").
		From("pvz")

	// Добавляем фильтрацию по датам, если указаны
	if params.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, params.StartDate)
		if err == nil {
			queryBuilder = queryBuilder.Where(squirrel.GtOrEq{"registration_date": startTime})
		}
	}

	if params.EndDate != "" {
		endTime, err := time.Parse(time.RFC3339, params.EndDate)
		if err == nil {
			queryBuilder = queryBuilder.Where(squirrel.LtOrEq{"registration_date": endTime})
		}
	}

	// Создаем отдельный запрос для подсчета
	countBuilder := q.sq.
		Select("COUNT(*)").
		From("pvz")

	// Копируем те же условия WHERE из основного запроса
	if params.StartDate != "" {
		startTime, err := time.Parse(time.RFC3339, params.StartDate)
		if err == nil {
			countBuilder = countBuilder.Where(squirrel.GtOrEq{"registration_date": startTime})
		}
	}

	if params.EndDate != "" {
		endTime, err := time.Parse(time.RFC3339, params.EndDate)
		if err == nil {
			countBuilder = countBuilder.Where(squirrel.LtOrEq{"registration_date": endTime})
		}
	}

	countQuery, countArgs, err := countBuilder.ToSql()

	if err != nil {
		return nil, 0, fmt.Errorf("failed to build count query: %w", err)
	}

	// Получаем общее количество записей
	var total int
	err = q.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pvz: %w", err)
	}

	// Добавляем пагинацию
	offset := (params.Page - 1) * params.Limit
	queryBuilder = queryBuilder.
		OrderBy("registration_date DESC").
		Limit(uint64(params.Limit)).
		Offset(uint64(offset))

	// Выполняем запрос с пагинацией
	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to build query: %w", err)
	}

	var pvzList []models.PVZ
	err = q.db.SelectContext(ctx, &pvzList, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get pvz list: %w", err)
	}

	return pvzList, total, nil
}
