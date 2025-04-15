package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"pvz-service/internal/models"
)

// Мок ProductQueries
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

// Мок ReceptionQueries
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

// Функция для настройки тестового окружения
func setupProductTest() (*gin.Engine, *MockProductQueries, *MockReceptionQueries) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	productQueries := new(MockProductQueries)
	receptionQueries := new(MockReceptionQueries)

	productHandler := NewProductHandler(productQueries, receptionQueries)

	// Тестовые маршруты
	r.POST("/products", func(c *gin.Context) {
		// Имитируем middleware, устанавливающий роль
		c.Set("userRole", "employee")
		productHandler.AddProduct(c)
	})

	r.POST("/pvz/:pvzId/delete_last_product", func(c *gin.Context) {
		// Имитируем middleware, устанавливающий роль
		c.Set("userRole", "employee")
		productHandler.DeleteLastProduct(c)
	})

	// Маршрут для тестирования доступа с неверной ролью
	r.POST("/products_moderator", func(c *gin.Context) {
		c.Set("userRole", "moderator")
		productHandler.AddProduct(c)
	})

	r.POST("/pvz/:pvzId/delete_last_product_moderator", func(c *gin.Context) {
		c.Set("userRole", "moderator")
		productHandler.DeleteLastProduct(c)
	})

	return r, productQueries, receptionQueries
}
