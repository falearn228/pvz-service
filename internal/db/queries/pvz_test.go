package queries

import (
	"context"
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

// setupPVZQueriesTest настраивает тестовое окружение для тестирования PVZQueries
func setupPVZQueriesTest(t *testing.T) (*PVZQueries, sqlmock.Sqlmock) {
	// Создаем новую мок-базу данных
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка при создании mock-базы данных: %v", err)
	}

	// Оборачиваем в sqlx
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	// Создаем экземпляр Database с моком
	dbInstance := &db.Database{DB: sqlxDB}

	// Создаем объект PVZQueries
	q := &PVZQueries{
		db: dbInstance,
		sq: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	return q, mock
}

func TestGetPVZList(t *testing.T) {
	// Настраиваем тестовое окружение
	pvzQueries, mock := setupPVZQueriesTest(t)

	t.Run("Базовое получение списка без фильтров", func(t *testing.T) {
		// Тестовые данные
		ctx := context.Background()
		params := models.PVZListQuery{
			Page:  1,
			Limit: 10,
		}

		// Подготавливаем тестовые ПВЗ
		expectedPVZs := []models.PVZ{
			{
				ID:               uuid.New().String(),
				RegistrationDate: time.Date(2025, 4, 1, 10, 0, 0, 0, time.UTC),
				City:             "Москва",
			},
			{
				ID:               uuid.New().String(),
				RegistrationDate: time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC),
				City:             "Санкт-Петербург",
			},
		}
		totalCount := 2

		// Настраиваем ожидание SQL-запроса для подсчета
		expectedCountSQL := `SELECT COUNT\(\*\) FROM pvz`
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(totalCount)
		mock.ExpectQuery(expectedCountSQL).WillReturnRows(countRows)

		// Настраиваем ожидание SQL-запроса для получения списка
		expectedSQL := `SELECT id, registration_date, city FROM pvz ORDER BY registration_date DESC LIMIT 10 OFFSET 0`
		rows := sqlmock.NewRows([]string{"id", "registration_date", "city"})
		for _, pvz := range expectedPVZs {
			rows.AddRow(pvz.ID, pvz.RegistrationDate, pvz.City)
		}
		mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

		// Вызываем тестируемый метод
		pvzList, total, err := pvzQueries.GetPVZList(ctx, params)

		// Проверяем результаты
		assert.NoError(t, err, "GetPVZList должен выполняться без ошибок")
		assert.Equal(t, totalCount, total, "Общее количество должно совпадать")
		assert.Equal(t, len(expectedPVZs), len(pvzList), "Количество ПВЗ в списке должно совпадать")

		for i, pvz := range pvzList {
			assert.Equal(t, expectedPVZs[i].ID, pvz.ID, "ID ПВЗ должен совпадать")
			assert.Equal(t, expectedPVZs[i].City, pvz.City, "Город ПВЗ должен совпадать")
			assert.True(t, expectedPVZs[i].RegistrationDate.Equal(pvz.RegistrationDate),
				"Дата регистрации должна совпадать")
		}

		// Проверяем, что все ожидания были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Не все ожидаемые запросы были выполнены")
	})

	t.Run("Получение списка с фильтрацией по датам", func(t *testing.T) {
		// Тестовые данные
		ctx := context.Background()
		startDate := "2025-03-01T00:00:00Z"
		endDate := "2025-04-01T00:00:00Z"
		params := models.PVZListQuery{
			Page:      1,
			Limit:     5,
			StartDate: startDate,
			EndDate:   endDate,
		}

		// Преобразуем строки дат в time.Time для проверки в SQL
		startTime, _ := time.Parse(time.RFC3339, startDate)
		endTime, _ := time.Parse(time.RFC3339, endDate)

		// Настраиваем ожидание SQL-запроса для подсчета с фильтрами
		expectedCountSQL := `SELECT COUNT\(\*\) FROM pvz WHERE registration_date >= \$1 AND registration_date <= \$2`
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(expectedCountSQL).
			WithArgs(startTime, endTime).
			WillReturnRows(countRows)

		// Настраиваем ожидание SQL-запроса для получения отфильтрованного списка
		expectedSQL := `SELECT id, registration_date, city FROM pvz WHERE registration_date >= \$1 AND registration_date <= \$2 ORDER BY registration_date DESC LIMIT 5 OFFSET 0`

		pvz := models.PVZ{
			ID:               uuid.New().String(),
			RegistrationDate: time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC),
			City:             "Санкт-Петербург",
		}

		rows := sqlmock.NewRows([]string{"id", "registration_date", "city"}).
			AddRow(pvz.ID, pvz.RegistrationDate, pvz.City)

		mock.ExpectQuery(expectedSQL).
			WithArgs(startTime, endTime).
			WillReturnRows(rows)

		// Вызываем тестируемый метод
		pvzList, total, err := pvzQueries.GetPVZList(ctx, params)

		// Проверяем результаты
		assert.NoError(t, err, "GetPVZList должен выполняться без ошибок")
		assert.Equal(t, 1, total, "Общее количество должно быть 1")
		assert.Equal(t, 1, len(pvzList), "Должен быть возвращен 1 ПВЗ")
		assert.Equal(t, pvz.ID, pvzList[0].ID, "ID ПВЗ должен совпадать")
		assert.Equal(t, pvz.City, pvzList[0].City, "Город ПВЗ должен совпадать")
		assert.True(t, pvz.RegistrationDate.Equal(pvzList[0].RegistrationDate),
			"Дата регистрации должна совпадать")

		// Проверяем, что все ожидания были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Не все ожидаемые запросы были выполнены")
	})

	t.Run("Ошибка при подсчете ПВЗ", func(t *testing.T) {
		// Тестовые данные
		ctx := context.Background()
		params := models.PVZListQuery{
			Page:  1,
			Limit: 10,
		}

		// Настраиваем ожидание SQL-запроса для подсчета, возвращающего ошибку
		expectedCountSQL := `SELECT COUNT\(\*\) FROM pvz`
		mock.ExpectQuery(expectedCountSQL).
			WillReturnError(errors.New("database error during count"))

		// Вызываем тестируемый метод
		pvzList, total, err := pvzQueries.GetPVZList(ctx, params)

		// Проверяем результаты
		assert.Error(t, err, "Должна возникнуть ошибка")
		assert.Nil(t, pvzList, "Список ПВЗ не должен быть возвращен при ошибке")
		assert.Equal(t, 0, total, "Общее количество должно быть 0 при ошибке")
		assert.Contains(t, err.Error(), "failed to count pvz", "Сообщение об ошибке должно содержать указанный текст")

		// Проверяем, что все ожидания были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Не все ожидаемые запросы были выполнены")
	})

	t.Run("Ошибка при получении списка ПВЗ", func(t *testing.T) {
		// Тестовые данные
		ctx := context.Background()
		params := models.PVZListQuery{
			Page:  1,
			Limit: 10,
		}

		// Настраиваем ожидание SQL-запроса для подсчета
		expectedCountSQL := `SELECT COUNT\(\*\) FROM pvz`
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
		mock.ExpectQuery(expectedCountSQL).WillReturnRows(countRows)

		// Настраиваем ожидание SQL-запроса для получения списка, возвращающего ошибку
		expectedSQL := `SELECT id, registration_date, city FROM pvz ORDER BY registration_date DESC LIMIT 10 OFFSET 0`
		mock.ExpectQuery(expectedSQL).
			WillReturnError(errors.New("database error during select"))

		// Вызываем тестируемый метод
		pvzList, total, err := pvzQueries.GetPVZList(ctx, params)

		// Проверяем результаты
		assert.Error(t, err, "Должна возникнуть ошибка")
		assert.Nil(t, pvzList, "Список ПВЗ не должен быть возвращен при ошибке")
		assert.Equal(t, 0, total, "Общее количество должно быть 0 при ошибке")
		assert.Contains(t, err.Error(), "failed to get pvz list", "Сообщение об ошибке должно содержать указанный текст")

		// Проверяем, что все ожидания были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Не все ожидаемые запросы были выполнены")
	})

	t.Run("Проверка пагинации", func(t *testing.T) {
		// Тестовые данные
		ctx := context.Background()
		params := models.PVZListQuery{
			Page:  3,
			Limit: 2,
		}
		totalCount := 7 // Всего 7 ПВЗ в базе

		// Настраиваем ожидание SQL-запроса для подсчета
		expectedCountSQL := `SELECT COUNT\(\*\) FROM pvz`
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(totalCount)
		mock.ExpectQuery(expectedCountSQL).WillReturnRows(countRows)

		// Настраиваем ожидание SQL-запроса для получения третьей страницы (offset = 4)
		expectedSQL := `SELECT id, registration_date, city FROM pvz ORDER BY registration_date DESC LIMIT 2 OFFSET 4`

		// На третьей странице должно быть 2 записи (из 7 всего)
		pvz1 := models.PVZ{
			ID:               uuid.New().String(),
			RegistrationDate: time.Date(2025, 2, 15, 12, 0, 0, 0, time.UTC),
			City:             "Казань",
		}
		pvz2 := models.PVZ{
			ID:               uuid.New().String(),
			RegistrationDate: time.Date(2025, 2, 10, 12, 0, 0, 0, time.UTC),
			City:             "Москва",
		}

		rows := sqlmock.NewRows([]string{"id", "registration_date", "city"}).
			AddRow(pvz1.ID, pvz1.RegistrationDate, pvz1.City).
			AddRow(pvz2.ID, pvz2.RegistrationDate, pvz2.City)

		mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

		// Вызываем тестируемый метод
		pvzList, total, err := pvzQueries.GetPVZList(ctx, params)

		// Проверяем результаты
		assert.NoError(t, err, "GetPVZList должен выполняться без ошибок")
		assert.Equal(t, totalCount, total, "Общее количество должно быть 7")
		assert.Equal(t, 2, len(pvzList), "Должно быть возвращено 2 ПВЗ")
		assert.Equal(t, pvz1.City, pvzList[0].City, "Город первого ПВЗ должен быть Казань")
		assert.Equal(t, pvz2.City, pvzList[1].City, "Город второго ПВЗ должен быть Москва")

		// Проверяем, что все ожидания были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Не все ожидаемые запросы были выполнены")
	})
}

func TestGetPVZListValidation(t *testing.T) {
	// Настраиваем тестовое окружение
	pvzQueries, mock := setupPVZQueriesTest(t)

	t.Run("Невалидный формат StartDate", func(t *testing.T) {
		// Тестовые данные с некорректным форматом даты
		ctx := context.Background()
		params := models.PVZListQuery{
			Page:      1,
			Limit:     10,
			StartDate: "2025/03/01", // Некорректный формат, не RFC3339
		}

		// Настраиваем ожидание SQL-запроса для подсчета (без фильтра по дате)
		expectedCountSQL := `SELECT COUNT\(\*\) FROM pvz`
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
		mock.ExpectQuery(expectedCountSQL).WillReturnRows(countRows)

		// Настраиваем ожидание SQL-запроса для получения списка (без фильтра по дате)
		expectedSQL := `SELECT id, registration_date, city FROM pvz ORDER BY registration_date DESC LIMIT 10 OFFSET 0`
		rows := sqlmock.NewRows([]string{"id", "registration_date", "city"})
		mock.ExpectQuery(expectedSQL).WillReturnRows(rows)

		// Вызываем тестируемый метод
		_, _, err := pvzQueries.GetPVZList(ctx, params)

		// Проверяем результаты - метод должен игнорировать невалидную дату и не возвращать ошибку
		assert.NoError(t, err, "GetPVZList должен игнорировать невалидный формат даты")

		// Проверяем, что все ожидания были выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "Не все ожидаемые запросы были выполнены")
	})
}
