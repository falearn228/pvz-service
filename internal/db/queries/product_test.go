package queries

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"

	"pvz-service/internal/db"
	"pvz-service/internal/models"
)

func setupProductQueriesTest(t *testing.T) (*ProductQueries, sqlmock.Sqlmock) {
	mockDB, mock, _ := sqlmock.New()
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	dbInstance := &db.Database{DB: sqlxDB}

	return &ProductQueries{
		db: dbInstance,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}, mock
}

func TestProductQueries_AddProduct(t *testing.T) {
	q, mock := setupProductQueriesTest(t)

	receptionID := uuid.New().String()
	productType := "электроника"
	now := time.Now().UTC()

	expectedSQL := `INSERT INTO product \(id,datetime,type,reception_id\) VALUES \(\$1,\$2,\$3,\$4\) RETURNING id, datetime, type, reception_id`
	t.Run("Успешное добавление товара", func(t *testing.T) {

		mock.ExpectQuery(expectedSQL).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), productType, receptionID).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "datetime", "type", "reception_id"}).
					AddRow(uuid.New().String(), now, productType, receptionID),
			)

		product, err := q.AddProduct(context.Background(), receptionID, productType)

		assert.NoError(t, err)
		assert.Equal(t, productType, product.Type)
		assert.Equal(t, receptionID, product.ReceptionID)
	})

	t.Run("Ошибка базы данных", func(t *testing.T) {
		mock.ExpectQuery(expectedSQL).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), productType, receptionID).
			WillReturnError(errors.New("database error"))

		product, err := q.AddProduct(context.Background(), receptionID, productType)

		assert.Error(t, err)
		assert.Nil(t, product)
	})
}

func TestProductQueries_GetLastProductFromReception(t *testing.T) {
	q, mock := setupProductQueriesTest(t)
	receptionID := uuid.New().String()

	expectedSQL := `SELECT id, datetime, type, reception_id FROM product WHERE reception_id = \$1 ORDER BY datetime DESC LIMIT 1`
	t.Run("Успешное получение последнего товара", func(t *testing.T) {
		product := models.Product{
			ID:          uuid.New().String(),
			Datetime:    time.Now(),
			Type:        "одежда",
			ReceptionID: receptionID,
		}

		mock.ExpectQuery(expectedSQL).
			WithArgs(receptionID).
			WillReturnRows(
				sqlmock.NewRows([]string{"id", "datetime", "type", "reception_id"}).
					AddRow(product.ID, product.Datetime, product.Type, product.ReceptionID),
			)

		result, err := q.GetLastProductFromReception(context.Background(), receptionID)

		assert.NoError(t, err)
		assert.Equal(t, product.ID, result.ID)
	})

	t.Run("Товары не найдены", func(t *testing.T) {
		mock.ExpectQuery(expectedSQL).
			WithArgs(receptionID).
			WillReturnError(sql.ErrNoRows)

		result, err := q.GetLastProductFromReception(context.Background(), receptionID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestProductQueries_DeleteProduct(t *testing.T) {
	q, mock := setupProductQueriesTest(t)
	productID := uuid.New().String()

	expectedSQL := `DELETE FROM product WHERE id = \$1`
	t.Run("Успешное удаление товара", func(t *testing.T) {

		mock.ExpectExec(expectedSQL).
			WithArgs(productID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := q.DeleteProduct(context.Background(), productID)

		assert.NoError(t, err)
	})

	t.Run("Товар не найден", func(t *testing.T) {
		mock.ExpectExec(expectedSQL).
			WithArgs(productID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := q.DeleteProduct(context.Background(), productID)

		assert.Error(t, err)
	})
}

func TestProductQueries_GetProductsByReception(t *testing.T) {
	q, mock := setupProductQueriesTest(t)
	receptionID := uuid.New().String()

	expectedSQL := `SELECT id, datetime, type, reception_id FROM product WHERE reception_id = \$1 ORDER BY datetime DESC`
	t.Run("Успешное получение товаров", func(t *testing.T) {
		products := []models.Product{
			{ID: uuid.New().String(), Datetime: time.Now(), Type: "электроника", ReceptionID: receptionID},
			{ID: uuid.New().String(), Datetime: time.Now().Add(-time.Hour), Type: "обувь", ReceptionID: receptionID},
		}

		rows := sqlmock.NewRows([]string{"id", "datetime", "type", "reception_id"})
		for _, p := range products {
			rows.AddRow(p.ID, p.Datetime, p.Type, p.ReceptionID)
		}

		mock.ExpectQuery(expectedSQL).
			WithArgs(receptionID).
			WillReturnRows(rows)

		result, err := q.GetProductsByReception(context.Background(), receptionID)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "электроника", result[0].Type)
	})

	t.Run("Ошибка выполнения запроса", func(t *testing.T) {
		mock.ExpectQuery(expectedSQL).
			WithArgs(receptionID).
			WillReturnError(errors.New("database error"))

		result, err := q.GetProductsByReception(context.Background(), receptionID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
