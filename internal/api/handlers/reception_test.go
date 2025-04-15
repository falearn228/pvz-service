package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"pvz-service/internal/models"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockReceptionQueries уже должен быть определен в других тестах
// Если нет, используем определение из предыдущих тестов

// Настройка тестового окружения
func setupReceptionTest() (*gin.Engine, *MockReceptionQueries) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	receptionQueries := new(MockReceptionQueries)

	receptionHandler := NewReceptionHandler(receptionQueries)

	// Настраиваем маршруты
	r.POST("/receptions", func(c *gin.Context) {
		c.Set("userRole", "employee") // Устанавливаем роль сотрудника
		receptionHandler.CreateReception(c)
	})

	r.POST("/pvz/:pvzId/close_last_reception", func(c *gin.Context) {
		receptionHandler.CloseLastReception(c)
	})

	return r, receptionQueries
}

// TestCreateReceptionSuccess проверяет успешное создание приёмки
func TestCreateReceptionSuccess(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"
	testReception := &models.Reception{
		ID:       "223e4567-e89b-12d3-a456-426614174000",
		DateTime: time.Date(2025, 4, 16, 4, 16, 0, 0, time.UTC),
		PvzID:    pvzID,
		Status:   "inprogress",
	}

	// Настраиваем моки
	receptionQueries.On("CheckOpenReception", mock.Anything, pvzID).Return(false, nil)
	receptionQueries.On("CreateReception", mock.Anything, pvzID).Return(testReception, nil)

	// Создаем запрос
	reqBody := models.CreateReceptionRequest{
		PvzID: pvzID,
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/receptions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Добавляем отладочный вывод
	t.Logf("Response status: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())

	// Проверяем ответ - должен быть статус 201 Created
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.ReceptionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, testReception.ID, response.ID)
	assert.Equal(t, testReception.PvzID, response.PvzID)
	assert.Equal(t, testReception.Status, response.Status)

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
}

// TestCreateReceptionForbidden проверяет запрет на создание приёмки не сотрудником
func TestCreateReceptionForbidden(t *testing.T) {
	// Создаем новый роутер с ролью модератора
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	receptionQueries := new(MockReceptionQueries)
	receptionHandler := NewReceptionHandler(receptionQueries)

	// Настраиваем маршрут с ролью модератора
	r.POST("/receptions", func(c *gin.Context) {
		c.Set("userRole", "moderator") // Устанавливаем роль модератора вместо сотрудника
		receptionHandler.CreateReception(c)
	})

	// Создаем запрос
	pvzID := "123e4567-e89b-12d3-a456-426614174000"
	reqBody := models.CreateReceptionRequest{
		PvzID: pvzID,
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/receptions", bytes.NewBuffer(jsonData))
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

	// Проверяем, что моки НЕ были вызваны
	receptionQueries.AssertNotCalled(t, "CheckOpenReception")
	receptionQueries.AssertNotCalled(t, "CreateReception")
}

// TestCreateReceptionAlreadyExists проверяет случай с уже существующей открытой приёмкой
func TestCreateReceptionAlreadyExists(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"

	// Настраиваем моки - уже есть открытая приёмка
	receptionQueries.On("CheckOpenReception", mock.Anything, pvzID).Return(true, nil)

	// Создаем запрос
	reqBody := models.CreateReceptionRequest{
		PvzID: pvzID,
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/receptions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "уже есть незакрытая приёмка")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
	receptionQueries.AssertNotCalled(t, "CreateReception")
}

// TestCreateReceptionInvalidRequest проверяет случай с некорректным запросом
func TestCreateReceptionInvalidRequest(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем запрос с некорректными данными (пустой PvzID)
	reqBody := map[string]string{
		"pvzId": "",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/receptions", bytes.NewBuffer(jsonData))
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

	// Проверяем, что моки НЕ были вызваны
	receptionQueries.AssertNotCalled(t, "CheckOpenReception")
	receptionQueries.AssertNotCalled(t, "CreateReception")
}

// TestCreateReceptionCheckError проверяет ошибку при проверке открытых приёмок
func TestCreateReceptionCheckError(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"

	// Настраиваем моки - ошибка при проверке
	receptionQueries.On("CheckOpenReception", mock.Anything, pvzID).Return(false, errors.New("database error"))

	// Создаем запрос
	reqBody := models.CreateReceptionRequest{
		PvzID: pvzID,
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/receptions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при проверке открытых приёмок")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
	receptionQueries.AssertNotCalled(t, "CreateReception")
}

// TestCreateReceptionCreateError проверяет ошибку при создании приёмки
func TestCreateReceptionCreateError(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"

	// Настраиваем моки - ошибка при создании
	receptionQueries.On("CheckOpenReception", mock.Anything, pvzID).Return(false, nil)
	receptionQueries.On("CreateReception", mock.Anything, pvzID).Return(nil, errors.New("database error"))

	// Создаем запрос
	reqBody := models.CreateReceptionRequest{
		PvzID: pvzID,
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/receptions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при создании приёмки")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
}

// TestCloseLastReceptionSuccess проверяет успешное закрытие приёмки
func TestCloseLastReceptionSuccess(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"
	receptionID := "223e4567-e89b-12d3-a456-426614174000"

	// Создаем тестовые приёмки
	openReception := &models.Reception{
		ID:       receptionID,
		DateTime: time.Date(2025, 4, 16, 4, 16, 0, 0, time.UTC),
		PvzID:    pvzID,
		Status:   "inprogress",
	}

	closedReception := &models.Reception{
		ID:       receptionID,
		DateTime: time.Date(2025, 4, 16, 4, 16, 0, 0, time.UTC),
		PvzID:    pvzID,
		Status:   "close",
	}

	// Настраиваем моки
	receptionQueries.On("GetLastOpenReception", mock.Anything, pvzID).Return(openReception, nil)
	receptionQueries.On("CloseReception", mock.Anything, receptionID).Return(closedReception, nil)

	// Создаем запрос
	req, _ := http.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Добавляем отладочный вывод
	t.Logf("Response status: %d", w.Code)
	t.Logf("Response body: %s", w.Body.String())

	// Проверяем ответ - должен быть статус 200 OK
	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ReceptionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, receptionID, response.ID)
	assert.Equal(t, pvzID, response.PvzID)
	assert.Equal(t, "close", response.Status)

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
}

// TestCloseLastReceptionEmptyPvzID проверяет случай с пустым ID ПВЗ
func TestCloseLastReceptionEmptyPvzID(t *testing.T) {
	// Создаем новый роутер
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.RemoveExtraSlash = true

	receptionQueries := new(MockReceptionQueries)
	receptionHandler := NewReceptionHandler(receptionQueries)

	// Настраиваем маршрут с пустым параметром pvzId
	r.POST("/pvz//close_last_reception", receptionHandler.CloseLastReception)

	// Создаем запрос с пустым pvzId
	req, _ := http.NewRequest("POST", "/pvz//close_last_reception", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Не указан ID ПВЗ", response.Message)

	// Проверяем, что моки НЕ были вызваны
	receptionQueries.AssertNotCalled(t, "GetLastOpenReception")
	receptionQueries.AssertNotCalled(t, "CloseReception")
}

// TestCloseLastReceptionNoOpenReception проверяет случай отсутствия открытой приёмки
func TestCloseLastReceptionNoOpenReception(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"

	// Настраиваем моки - нет открытой приёмки
	receptionQueries.On("GetLastOpenReception", mock.Anything, pvzID).Return(nil, errors.New("no open reception found"))

	// Создаем запрос
	req, _ := http.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при получении приёмки")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
	receptionQueries.AssertNotCalled(t, "CloseReception")
}

// TestCloseLastReceptionCloseError проверяет ошибку при закрытии приёмки
func TestCloseLastReceptionCloseError(t *testing.T) {
	r, receptionQueries := setupReceptionTest()

	// Создаем тестовые данные
	pvzID := "123e4567-e89b-12d3-a456-426614174000"
	receptionID := "223e4567-e89b-12d3-a456-426614174000"

	// Создаем тестовую приёмку
	openReception := &models.Reception{
		ID:       receptionID,
		DateTime: time.Date(2025, 4, 16, 4, 16, 0, 0, time.UTC),
		PvzID:    pvzID,
		Status:   "inprogress",
	}

	// Настраиваем моки - ошибка при закрытии
	receptionQueries.On("GetLastOpenReception", mock.Anything, pvzID).Return(openReception, nil)
	receptionQueries.On("CloseReception", mock.Anything, receptionID).Return(nil, errors.New("database error"))

	// Создаем запрос
	req, _ := http.NewRequest("POST", "/pvz/"+pvzID+"/close_last_reception", nil)

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при закрытии приёмки")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
}
