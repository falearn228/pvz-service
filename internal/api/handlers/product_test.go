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

// MockProductQueries мокирует запросы для работы с товарами
type MockProductQueries struct {
	mock.Mock
}

func (m *MockProductQueries) AddProduct(ctx context.Context, receptionID, productType string) (*models.Product, error) {
	args := m.Called(ctx, receptionID, productType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductQueries) GetLastProductFromReception(ctx context.Context, receptionID string) (*models.Product, error) {
	args := m.Called(ctx, receptionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Product), args.Error(1)
}

func (m *MockProductQueries) DeleteProduct(ctx context.Context, productID string) error {
	args := m.Called(ctx, productID)
	return args.Error(0)
}

// MockReceptionQueries мокирует запросы для работы с приёмками
type MockReceptionQueries struct {
	mock.Mock
}

func (m *MockReceptionQueries) GetLastOpenReception(ctx context.Context, pvzID string) (*models.Reception, error) {
	args := m.Called(ctx, pvzID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Reception), args.Error(1)
}

// Настройка тестового окружения
func setupProductTest() (*gin.Engine, *MockProductQueries, *MockReceptionQueries) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	productQueries := new(MockProductQueries)
	receptionQueries := new(MockReceptionQueries)

	productHandler := NewProductHandler(productQueries, receptionQueries)

	// Создаем группу маршрутов с middleware для установки роли пользователя
	authorized := r.Group("/")
	authorized.Use(func(c *gin.Context) {
		c.Set("userRole", "employee") // По умолчанию устанавливаем роль employee
		c.Next()
	})

	authorized.POST("/products", productHandler.AddProduct)
	authorized.POST("/pvz/:pvzId/delete_last_product", productHandler.DeleteLastProduct)

	return r, productQueries, receptionQueries
}

// Функция для установки роли пользователя в тестах
func setUserRole(r *gin.Engine, role string) {
	r.Use(func(c *gin.Context) {
		c.Set("userRole", role)
		c.Next()
	})
}

// TestAddProductSuccess проверяет успешное добавление товара
func TestAddProductSuccess(t *testing.T) {
	r, productQueries, receptionQueries := setupProductTest()

	// Создаем тестовые данные
	testReception := &models.Reception{
		ID:       "reception-uuid",
		DateTime: time.Now(),
		PvzID:    "pvz-uuid",
		Status:   "in_progress",
	}

	testProduct := &models.Product{
		ID:                "product-uuid",
		ReceptionDatetime: time.Now(),
		Type:              "электроника",
		ReceptionID:       "reception-uuid",
	}

	// Настраиваем моки
	receptionQueries.On("GetLastOpenReception", mock.Anything, "pvz-uuid").Return(testReception, nil)
	productQueries.On("AddProduct", mock.Anything, "reception-uuid", "электроника").Return(testProduct, nil)

	// Создаем запрос
	reqBody := models.CreateProductRequest{
		Type:  "электроника",
		PvzID: "pvz-uuid",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ
	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.ProductResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "product-uuid", response.ID)
	assert.Equal(t, "электроника", response.Type)
	assert.Equal(t, "reception-uuid", response.ReceptionID)

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
	productQueries.AssertExpectations(t)
}

// TestAddProductForbidden проверяет запрет на добавление товара не сотрудником
func TestAddProductForbidden(t *testing.T) {
	r, _, _ := setupProductTest()

	// Создаем новый роутер с ролью модератора
	moderatorRouter := gin.Default()
	moderatorRouter.Use(func(c *gin.Context) {
		c.Set("userRole", "moderator") // Устанавливаем роль модератора
		c.Next()
	})

	// Регистрируем обработчик
	productHandler := NewProductHandler(new(MockProductQueries), new(MockReceptionQueries))
	moderatorRouter.POST("/products", productHandler.AddProduct)

	// Создаем запрос
	reqBody := models.CreateProductRequest{
		Type:  "электроника",
		PvzID: "pvz-uuid",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	moderatorRouter.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 403 Forbidden
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Доступ запрещен")
}

// TestAddProductNoOpenReception проверяет случай отсутствия открытой приёмки
func TestAddProductNoOpenReception(t *testing.T) {
	r, _, receptionQueries := setupProductTest()

	// Настраиваем моки - нет открытой приёмки
	receptionQueries.On("GetLastOpenReception", mock.Anything, "pvz-uuid").
		Return(nil, errors.New("no open reception found"))

	// Создаем запрос
	reqBody := models.CreateProductRequest{
		Type:  "электроника",
		PvzID: "pvz-uuid",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Нет активной приёмки для данного ПВЗ")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
}

// TestAddProductClosedReception проверяет случай с закрытой приёмкой
func TestAddProductClosedReception(t *testing.T) {
	r, _, receptionQueries := setupProductTest()

	// Создаем тестовые данные - закрытая приёмка
	testReception := &models.Reception{
		ID:       "reception-uuid",
		DateTime: time.Now(),
		PvzID:    "pvz-uuid",
		Status:   "close", // Закрытая приёмка
	}

	// Настраиваем моки
	receptionQueries.On("GetLastOpenReception", mock.Anything, "pvz-uuid").Return(testReception, nil)

	// Создаем запрос
	reqBody := models.CreateProductRequest{
		Type:  "электроника",
		PvzID: "pvz-uuid",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Приёмка уже закрыта", response.Message)

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
}

// TestAddProductError проверяет ошибку при добавлении товара
func TestAddProductError(t *testing.T) {
	r, productQueries, receptionQueries := setupProductTest()

	// Создаем тестовые данные
	testReception := &models.Reception{
		ID:       "reception-uuid",
		DateTime: time.Now(),
		PvzID:    "pvz-uuid",
		Status:   "in_progress",
	}

	// Настраиваем моки
	receptionQueries.On("GetLastOpenReception", mock.Anything, "pvz-uuid").Return(testReception, nil)
	productQueries.On("AddProduct", mock.Anything, "reception-uuid", "электроника").
		Return(nil, errors.New("database error"))

	// Создаем запрос
	reqBody := models.CreateProductRequest{
		Type:  "электроника",
		PvzID: "pvz-uuid",
	}
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/products", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	// Выполняем запрос
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверяем ответ - должен быть статус 500 Internal Server Error
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response models.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Message, "Ошибка при добавлении товара")

	// Проверяем, что моки были вызваны с правильными аргументами
	receptionQueries.AssertExpectations(t)
	productQueries.AssertExpectations(t)
}
