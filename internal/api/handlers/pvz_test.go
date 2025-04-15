package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pvz-service/internal/models"
)

// MockPVZQueries мокирует запросы для работы с ПВЗ
type MockPVZQueries struct {
	mock.Mock
}

func (m *MockReceptionQueries) CheckOpenReception(ctx context.Context, pvzID string) (bool, error) {
	args := m.Called(ctx, pvzID)
	return args.Bool(0), args.Error(1)
}

func (m *MockReceptionQueries) CreateReception(ctx context.Context, pvzID string) (*models.Reception, error) {
	args := m.Called(ctx, pvzID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reception), args.Error(1)
}

func (m *MockReceptionQueries) CloseReception(ctx context.Context, receptionID string) (*models.Reception, error) {
	args := m.Called(ctx, receptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reception), args.Error(1)
}

func (m *MockReceptionQueries) GetReceptionsByPVZ(ctx context.Context, pvzID string) ([]models.Reception, error) {
	args := m.Called(ctx, pvzID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Reception), args.Error(1)
}

func (m *MockPVZQueries) CreatePVZ(ctx context.Context, city string) (*models.PVZ, error) {
	args := m.Called(ctx, city)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PVZ), args.Error(1)
}

func (m *MockPVZQueries) GetPVZList(ctx context.Context, params models.PVZListQuery) ([]models.PVZ, int, error) {
	args := m.Called(ctx, params)

	var pvzList []models.PVZ
	if args.Get(0) != nil {
		pvzList = args.Get(0).([]models.PVZ)
	}

	return pvzList, args.Int(1), args.Error(2)
}

// Настройка тестового окружения
func setupPVZTest() (*gin.Engine, *MockPVZQueries, *MockReceptionQueries, *MockProductQueries) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	pvzQueries := new(MockPVZQueries)
	receptionQueries := new(MockReceptionQueries)
	productQueries := new(MockProductQueries)

	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)

	// Настраиваем маршрут для создания ПВЗ
	// В реальном приложении здесь должна быть проверка роли "moderator"
	r.POST("/pvz", func(c *gin.Context) {
		c.Set("userRole", "moderator") // Устанавливаем роль модератора
		pvzHandler.CreatePVZ(c)
	})

	return r, pvzQueries, receptionQueries, productQueries
}

// TestCreatePVZSuccess проверяет успешное создание ПВЗ
func TestCreatePVZSuccess(t *testing.T) {
	r, pvzQueries, _, _ := setupPVZTest()

	// Создаем тестовые данные
	testPVZ := &models.PVZ{
		ID:               "123e4567-e89b-12d3-a456-426614174000",
		RegistrationDate: time.Now(),
		City:             "Москва",
	}

	// Настраиваем моки
	pvzQueries.On("CreatePVZ", mock.Anything, "Москва").Return(testPVZ, nil)

	// Создаем запрос
	reqBody := models.CreatePVZRequest{
		City: "Москва",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Добавляем отладочный вывод
	t.Logf("Response status: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())

	// Проверяем ответ - должен быть статус 201 Created
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.PVZResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testPVZ.ID, response.ID)
	assert.Equal(t, testPVZ.City, response.City)

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
}

// TestCreatePVZInvalidRequest проверяет случай с некорректным запросом
func TestCreatePVZInvalidRequest(t *testing.T) {
	r, _, _, _ := setupPVZTest()

	// Создаем запрос с некорректными данными (город не из списка допустимых)
	reqBody := map[string]string{
		"city": "Новосибирск", // Город не входит в список допустимых (Москва, Санкт-Петербург, Казань)
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}

// TestCreatePVZEmptyCity проверяет случай с пустым городом
func TestCreatePVZEmptyCity(t *testing.T) {
	r, _, _, _ := setupPVZTest()

	// Создаем запрос с пустым городом
	reqBody := map[string]string{
		"city": "",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}

// TestCreatePVZDatabaseError проверяет случай с ошибкой базы данных
func TestCreatePVZDatabaseError(t *testing.T) {
	r, pvzQueries, _, _ := setupPVZTest()

	// Настраиваем моки - ошибка при создании ПВЗ
	pvzQueries.On("CreatePVZ", mock.Anything, "Москва").Return(nil, errors.New("database error"))

	// Создаем запрос
	reqBody := models.CreatePVZRequest{
		City: "Москва",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при создании ПВЗ")

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
}

// TestCreatePVZMissingCity проверяет случай с отсутствующим полем city
func TestCreatePVZMissingCity(t *testing.T) {
	r, _, _, _ := setupPVZTest()

	// Создаем запрос без поля city
	reqBody := map[string]string{}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}

// TestCreatePVZInvalidJSON проверяет случай с некорректным JSON
func TestCreatePVZInvalidJSON(t *testing.T) {
	r, _, _, _ := setupPVZTest()

	// Создаем запрос с некорректным JSON
	invalidJSON := []byte(`{"city": "Москва"`) // Отсутствует закрывающая скобка
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверный запрос")
}

// TestCreatePVZForbidden проверяет запрет на создание ПВЗ не модератором
func TestCreatePVZForbidden(t *testing.T) {
	// Создаем новый роутер с ролью employee
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	pvzQueries := new(MockPVZQueries)
	receptionQueries := new(MockReceptionQueries)
	productQueries := new(MockProductQueries)

	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)

	// Настраиваем маршрут с ролью employee
	r.POST("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee") // Устанавливаем роль employee вместо moderator
		pvzHandler.CreatePVZ(c)
	})

	// Создаем запрос
	reqBody := models.CreatePVZRequest{
		City: "Москва",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/pvz", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Доступ запрещен")
}

// TestGetPVZListSuccess проверяет успешное получение списка ПВЗ
func TestGetPVZListSuccess(t *testing.T) {
	r, pvzQueries, receptionQueries, productQueries := setupPVZTest()
	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)
	// Создаем тестовые данные
	testPVZList := []models.PVZ{
		{
			ID:               "123e4567-e89b-12d3-a456-426614174000",
			RegistrationDate: time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC),
			City:             "Москва",
		},
		{
			ID:               "223e4567-e89b-12d3-a456-426614174000",
			RegistrationDate: time.Date(2025, 3, 10, 10, 0, 0, 0, time.UTC),
			City:             "Санкт-Петербург",
		},
	}

	// Создаем тестовые приёмки для первого ПВЗ
	testReceptions1 := []models.Reception{
		{
			ID:       "323e4567-e89b-12d3-a456-426614174000",
			DateTime: time.Date(2025, 4, 1, 10, 0, 0, 0, time.UTC),
			PvzID:    "123e4567-e89b-12d3-a456-426614174000",
			Status:   "inprogress",
		},
	}

	// Создаем тестовые товары для первой приёмки
	testProducts1 := []models.Product{
		{
			ID:          "423e4567-e89b-12d3-a456-426614174000",
			Datetime:    time.Date(2025, 4, 1, 11, 0, 0, 0, time.UTC),
			Type:        "электроника",
			ReceptionID: "323e4567-e89b-12d3-a456-426614174000",
		},
	}

	// Создаем тестовые приёмки для второго ПВЗ
	testReceptions2 := []models.Reception{
		{
			ID:       "523e4567-e89b-12d3-a456-426614174000",
			DateTime: time.Date(2025, 3, 20, 10, 0, 0, 0, time.UTC),
			PvzID:    "223e4567-e89b-12d3-a456-426614174000",
			Status:   "close",
		},
	}

	// Создаем тестовые товары для второй приёмки
	testProducts2 := []models.Product{
		{
			ID:          "623e4567-e89b-12d3-a456-426614174000",
			Datetime:    time.Date(2025, 3, 20, 11, 0, 0, 0, time.UTC),
			Type:        "одежда",
			ReceptionID: "523e4567-e89b-12d3-a456-426614174000",
		},
	}

	// Параметры запроса
	params := models.PVZListQuery{
		StartDate: "2025-03-01T00:00:00Z",
		EndDate:   "2025-04-15T00:00:00Z",
		Page:      1,
		Limit:     10,
	}

	// Настраиваем моки
	pvzQueries.On("GetPVZList", mock.Anything, params).Return(testPVZList, 2, nil)
	receptionQueries.On("GetReceptionsByPVZ", mock.Anything, "123e4567-e89b-12d3-a456-426614174000").Return(testReceptions1, nil)
	receptionQueries.On("GetReceptionsByPVZ", mock.Anything, "223e4567-e89b-12d3-a456-426614174000").Return(testReceptions2, nil)
	productQueries.On("GetProductsByReception", mock.Anything, "323e4567-e89b-12d3-a456-426614174000").Return(testProducts1, nil)
	productQueries.On("GetProductsByReception", mock.Anything, "523e4567-e89b-12d3-a456-426614174000").Return(testProducts2, nil)

	// Настраиваем маршрут для получения списка ПВЗ
	r.GET("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee") // Роль не важна для этого эндпоинта
		pvzHandler.GetPVZList(c)
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/pvz?startDate=2025-03-01T00:00:00Z&endDate=2025-04-15T00:00:00Z&page=1&limit=10", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Добавляем отладочный вывод
	t.Logf("Response status: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем заголовок X-Total-Count
	assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

	// Проверяем содержимое ответа
	var response []models.PVZWithReceptionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Проверяем количество ПВЗ в ответе
	assert.Equal(t, 2, len(response))

	// Проверяем данные первого ПВЗ
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", response[0].PVZ.ID)
	assert.Equal(t, "Москва", response[0].PVZ.City)

	// Проверяем приёмки первого ПВЗ
	assert.Equal(t, 1, len(response[0].Receptions))
	assert.Equal(t, "323e4567-e89b-12d3-a456-426614174000", response[0].Receptions[0].Reception.ID)
	assert.Equal(t, "inprogress", response[0].Receptions[0].Reception.Status)

	// Проверяем товары в первой приёмке
	assert.Equal(t, 1, len(response[0].Receptions[0].Products))
	assert.Equal(t, "423e4567-e89b-12d3-a456-426614174000", response[0].Receptions[0].Products[0].ID)
	assert.Equal(t, "электроника", response[0].Receptions[0].Products[0].Type)

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
	receptionQueries.AssertExpectations(t)
	productQueries.AssertExpectations(t)
}

// TestGetPVZListEmptyResult проверяет получение пустого списка ПВЗ
func TestGetPVZListEmptyResult(t *testing.T) {
	r, pvzQueries, receptionQueries, productQueries := setupPVZTest()
	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)
	// Параметры запроса
	params := models.PVZListQuery{
		StartDate: "2026-01-01T00:00:00Z", // Будущая дата, когда нет ПВЗ
		EndDate:   "2026-12-31T23:59:59Z",
		Page:      1,
		Limit:     10,
	}

	// Настраиваем моки - пустой список
	pvzQueries.On("GetPVZList", mock.Anything, params).Return([]models.PVZ{}, 0, nil)

	// Настраиваем маршрут для получения списка ПВЗ
	r.GET("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee")
		pvzHandler.GetPVZList(c)
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/pvz?startDate=2026-01-01T00:00:00Z&endDate=2026-12-31T23:59:59Z&page=1&limit=10", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем заголовок X-Total-Count
	assert.Equal(t, "0", w.Header().Get("X-Total-Count"))

	// Проверяем содержимое ответа - пустой массив
	var response []models.PVZWithReceptionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(response))

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
}

// TestGetPVZListPagination проверяет работу пагинации
func TestGetPVZListPagination(t *testing.T) {
	r, pvzQueries, receptionQueries, productQueries := setupPVZTest()
	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)

	// Создаем тестовые данные - только один ПВЗ на второй странице
	testPVZList := []models.PVZ{
		{
			ID:               "323e4567-e89b-12d3-a456-426614174000",
			RegistrationDate: time.Date(2025, 2, 15, 10, 0, 0, 0, time.UTC),
			City:             "Казань",
		},
	}

	// Создаем тестовые приёмки
	testReceptions := []models.Reception{}

	// Параметры запроса - вторая страница
	params := models.PVZListQuery{
		Page:  2,
		Limit: 1, // Один элемент на странице
	}

	// Настраиваем моки
	pvzQueries.On("GetPVZList", mock.Anything, params).Return(testPVZList, 3, nil) // Всего 3 элемента
	receptionQueries.On("GetReceptionsByPVZ", mock.Anything, "323e4567-e89b-12d3-a456-426614174000").Return(testReceptions, nil)

	// Настраиваем маршрут для получения списка ПВЗ
	r.GET("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee")
		pvzHandler.GetPVZList(c)
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/pvz?page=2&limit=1", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем заголовок X-Total-Count
	assert.Equal(t, "3", w.Header().Get("X-Total-Count"))

	// Проверяем содержимое ответа - один элемент
	var response []models.PVZWithReceptionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(response))
	assert.Equal(t, "323e4567-e89b-12d3-a456-426614174000", response[0].PVZ.ID)
	assert.Equal(t, "Казань", response[0].PVZ.City)

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
	receptionQueries.AssertExpectations(t)
}

// TestGetPVZListInvalidParams проверяет обработку некорректных параметров
func TestGetPVZListInvalidParams(t *testing.T) {
	r, pvzQueries, receptionQueries, productQueries := setupPVZTest()
	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)

	// Параметры запроса с некорректными значениями
	params := models.PVZListQuery{
		Page:  0,  // Некорректное значение (должно быть >= 1)
		Limit: 50, // Некорректное значение (должно быть <= 30)
	}

	// Настраиваем моки - ошибка валидации
	pvzQueries.On("GetPVZList", mock.Anything, params).Return(nil, 0, errors.New("invalid pagination parameters"))

	// Настраиваем маршрут для получения списка ПВЗ
	r.GET("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee")
		pvzHandler.GetPVZList(c)
	})

	// Создаем запрос с некорректными параметрами
	req, _ := http.NewRequest("GET", "/pvz?page=0&limit=50", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Неверные параметры запроса")

	// Не устанавливаем ложные ожидания для pvzQueries.GetPVZList
	// поскольку он не будет вызван из-за сбоя проверки (так как БД функция вызовится
	// только, если пройдет валидация)
	pvzQueries.AssertNotCalled(t, "GetPVZList")
}

// TestGetPVZListDatabaseError проверяет обработку ошибки базы данных
func TestGetPVZListDatabaseError(t *testing.T) {
	r, pvzQueries, receptionQueries, productQueries := setupPVZTest()
	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)

	// Параметры запроса
	params := models.PVZListQuery{
		Page:  1,
		Limit: 10,
	}

	// Настраиваем моки - ошибка базы данных
	pvzQueries.On("GetPVZList", mock.Anything, params).Return(nil, 0, errors.New("database connection error"))

	// Настраиваем маршрут для получения списка ПВЗ
	r.GET("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee")
		pvzHandler.GetPVZList(c)
	})

	// Создаем запрос
	req, _ := http.NewRequest("GET", "/pvz?page=1&limit=10", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при получении списка ПВЗ")

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
}

// TestGetPVZListDateFilter проверяет фильтрацию по датам
func TestGetPVZListDateFilter(t *testing.T) {
	r, pvzQueries, receptionQueries, productQueries := setupPVZTest()
	pvzHandler := NewPVZHandler(pvzQueries, receptionQueries, productQueries)

	// Создаем тестовые данные - ПВЗ в заданном диапазоне дат
	testPVZList := []models.PVZ{
		{
			ID:               "123e4567-e89b-12d3-a456-426614174000",
			RegistrationDate: time.Date(2025, 3, 15, 10, 0, 0, 0, time.UTC),
			City:             "Москва",
		},
	}

	// Создаем тестовые приёмки
	testReceptions := []models.Reception{}

	// Параметры запроса с фильтрацией по датам
	params := models.PVZListQuery{
		StartDate: "2025-03-01T00:00:00Z",
		EndDate:   "2025-03-31T23:59:59Z",
		Page:      1,
		Limit:     10,
	}

	// Настраиваем моки
	pvzQueries.On("GetPVZList", mock.Anything, params).Return(testPVZList, 1, nil)
	receptionQueries.On("GetReceptionsByPVZ", mock.Anything, "123e4567-e89b-12d3-a456-426614174000").Return(testReceptions, nil)

	// Настраиваем маршрут для получения списка ПВЗ
	r.GET("/pvz", func(c *gin.Context) {
		c.Set("userRole", "employee")
		pvzHandler.GetPVZList(c)
	})

	// Создаем запрос с фильтрацией по датам
	req, _ := http.NewRequest("GET", "/pvz?startDate=2025-03-01T00:00:00Z&endDate=2025-03-31T23:59:59Z&page=1&limit=10", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	// Проверяем заголовок X-Total-Count
	assert.Equal(t, "1", w.Header().Get("X-Total-Count"))

	// Проверяем содержимое ответа - один элемент
	var response []models.PVZWithReceptionsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(response))
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", response[0].PVZ.ID)
	assert.Equal(t, "Москва", response[0].PVZ.City)

	// Проверяем, что моки были вызваны с правильными аргументами
	pvzQueries.AssertExpectations(t)
	receptionQueries.AssertExpectations(t)
}
