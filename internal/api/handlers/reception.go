package handlers

import (
	"net/http"

	"pvz-service/internal/db/queries"
	"pvz-service/internal/models"

	"github.com/gin-gonic/gin"
)

// ReceptionHandler содержит обработчики для работы с приёмками товаров
type ReceptionHandler struct {
	receptionQueries *queries.ReceptionQueries
}

// NewReceptionHandler создает новый экземпляр ReceptionHandler
func NewReceptionHandler(receptionQueries *queries.ReceptionQueries) *ReceptionHandler {
	return &ReceptionHandler{
		receptionQueries: receptionQueries,
	}
}

// CreateReception обрабатывает запрос на создание приёмки товаров
func (h *ReceptionHandler) CreateReception(c *gin.Context) {
	// Проверяем, что пользователь - сотрудник
	userRole, _ := c.Get("userRole")
	if userRole != "employee" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Message: "Доступ запрещен: только сотрудники могут создавать приёмки",
		})
		return
	}

	var req models.CreateReceptionRequest

	// Проверяем запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверный запрос: " + err.Error(),
		})
		return
	}

	// Проверяем, есть ли уже открытая приёмка для этого ПВЗ
	hasOpen, err := h.receptionQueries.CheckOpenReception(c.Request.Context(), req.PvzID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при проверке открытых приёмок: " + err.Error(),
		})
		return
	}

	if hasOpen {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Для данного ПВЗ уже есть незакрытая приёмка",
		})
		return
	}

	// Создаем приёмку
	reception, err := h.receptionQueries.CreateReception(c.Request.Context(), req.PvzID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при создании приёмки: " + err.Error(),
		})
		return
	}

	// Возвращаем данные созданной приёмки
	c.JSON(http.StatusCreated, models.ReceptionResponse{
		ID:       reception.ID,
		DateTime: reception.DateTime,
		PvzID:    reception.PvzID,
		Status:   reception.Status,
	})
}

// CloseLastReception обрабатывает запрос на закрытие последней открытой приёмки товаров
func (h *ReceptionHandler) CloseLastReception(c *gin.Context) {
	pvzID := c.Param("pvzId")

	// Проверяем, что pvzId указан
	if pvzID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Не указан ID ПВЗ",
		})
		return
	}

	// Получаем последнюю открытую приёмку
	reception, err := h.receptionQueries.GetLastOpenReception(c.Request.Context(), pvzID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Ошибка при получении приёмки: " + err.Error(),
		})
		return
	}

	// Закрываем приёмку
	closedReception, err := h.receptionQueries.CloseReception(c.Request.Context(), reception.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при закрытии приёмки: " + err.Error(),
		})
		return
	}

	// Возвращаем данные закрытой приёмки
	c.JSON(http.StatusOK, models.ReceptionResponse{
		ID:       closedReception.ID,
		DateTime: closedReception.DateTime,
		PvzID:    closedReception.PvzID,
		Status:   closedReception.Status,
	})
}
