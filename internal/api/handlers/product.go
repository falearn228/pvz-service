package handlers

import (
	"net/http"

	"pvz-service/internal/db/queries"
	"pvz-service/internal/models"

	"github.com/gin-gonic/gin"
)

// ProductHandler содержит обработчики для работы с товарами
type ProductHandler struct {
	productQueries   *queries.ProductQueries
	receptionQueries *queries.ReceptionQueries
}

// NewProductHandler создает новый экземпляр ProductHandler
func NewProductHandler(productQueries *queries.ProductQueries, receptionQueries *queries.ReceptionQueries) *ProductHandler {
	return &ProductHandler{
		productQueries:   productQueries,
		receptionQueries: receptionQueries,
	}
}

// AddProduct обрабатывает запрос на добавление товара в приёмку
func (h *ProductHandler) AddProduct(c *gin.Context) {
	// Проверяем, что пользователь - сотрудник
	userRole, _ := c.Get("userRole")
	if userRole != "employee" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Message: "Доступ запрещен: только сотрудники могут добавлять товары",
		})
		return
	}

	var req models.CreateProductRequest

	// Проверяем запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверный запрос: " + err.Error(),
		})
		return
	}

	// Получаем последнюю открытую приёмку для ПВЗ
	reception, err := h.receptionQueries.GetLastOpenReception(c.Request.Context(), req.PvzID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Нет активной приёмки для данного ПВЗ: " + err.Error(),
		})
		return
	}

	// Проверяем, что статус приёмки - "in_progress"
	if reception.Status != "in_progress" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Приёмка уже закрыта",
		})
		return
	}

	// Добавляем товар
	product, err := h.productQueries.AddProduct(c.Request.Context(), reception.ID, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при добавлении товара: " + err.Error(),
		})
		return
	}

	// Возвращаем данные добавленного товара
	c.JSON(http.StatusCreated, models.ProductResponse{
		ID:          product.ID,
		DateTime:    product.ReceptionDatetime,
		Type:        product.Type,
		ReceptionID: product.ReceptionID,
	})
}

// DeleteLastProduct обрабатывает запрос на удаление последнего добавленного товара
func (h *ProductHandler) DeleteLastProduct(c *gin.Context) {
	// Проверяем, что пользователь - сотрудник
	userRole, _ := c.Get("userRole")
	if userRole != "employee" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Message: "Доступ запрещен: только сотрудники могут удалять товары",
		})
		return
	}

	pvzID := c.Param("pvzId")

	// Проверяем, что pvzId указан
	if pvzID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Не указан ID ПВЗ",
		})
		return
	}

	// Получаем последнюю открытую приёмку для ПВЗ
	reception, err := h.receptionQueries.GetLastOpenReception(c.Request.Context(), pvzID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Нет активной приёмки для данного ПВЗ: " + err.Error(),
		})
		return
	}

	// Проверяем, что статус приёмки - "in_progress"
	if reception.Status != "in_progress" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Приёмка уже закрыта",
		})
		return
	}

	// Получаем последний добавленный товар
	product, err := h.productQueries.GetLastProductFromReception(c.Request.Context(), reception.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Нет товаров для удаления в данной приёмке: " + err.Error(),
		})
		return
	}

	// Удаляем товар
	err = h.productQueries.DeleteProduct(c.Request.Context(), product.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при удалении товара: " + err.Error(),
		})
		return
	}

	// Возвращаем успешный ответ
	c.Status(http.StatusOK)
}
