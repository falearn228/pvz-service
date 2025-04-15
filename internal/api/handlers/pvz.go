// internal/api/handlers/pvz.go
package handlers

import (
	"fmt"
	"net/http"

	"pvz-service/internal/db/queries"
	"pvz-service/internal/models"

	"github.com/gin-gonic/gin"
)

// PVZHandler содержит обработчики для работы с ПВЗ
type PVZHandler struct {
	pvzQueries       queries.PVZQueriesInterface
	receptionQueries queries.ReceptionQueriesInterface
	productQueries   queries.ProductQueriesInterface
}

// NewPVZHandler создает новый экземпляр PVZHandler
func NewPVZHandler(pvzQueries queries.PVZQueriesInterface, receptionQueries queries.ReceptionQueriesInterface, productQueries queries.ProductQueriesInterface) *PVZHandler {
	return &PVZHandler{
		pvzQueries:       pvzQueries,
		receptionQueries: receptionQueries,
		productQueries:   productQueries,
	}
}

// CreatePVZ обрабатывает запрос на создание ПВЗ
func (h *PVZHandler) CreatePVZ(c *gin.Context) {
	var req models.CreatePVZRequest

	// Проверяем запрос
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверный запрос: " + err.Error(),
		})
		return
	}

	userRole, _ := c.Get("userRole")
	if userRole != "moderator" {
		c.JSON(http.StatusForbidden, models.ErrorResponse{
			Message: "Доступ запрещен: только модераторы могут создавать ПВЗ",
		})
		return
	}

	// Создаем ПВЗ
	pvz, err := h.pvzQueries.CreatePVZ(c.Request.Context(), req.City)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при создании ПВЗ: " + err.Error(),
		})
		return
	}

	// Возвращаем данные созданного ПВЗ
	c.JSON(http.StatusCreated, models.PVZResponse{
		ID:               pvz.ID,
		RegistrationDate: pvz.RegistrationDate,
		City:             pvz.City,
	})
}

// GetPVZList обрабатывает запрос на получение списка ПВЗ с фильтрацией и пагинацией
func (h *PVZHandler) GetPVZList(c *gin.Context) {
	var query models.PVZListQuery

	// Устанавливаем значения по умолчанию
	query.Page = 1
	query.Limit = 10

	// Извлекаем параметры запроса
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Message: "Неверные параметры запроса: " + err.Error(),
		})
		return
	}

	// Получаем список ПВЗ
	pvzList, total, err := h.pvzQueries.GetPVZList(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Message: "Ошибка при получении списка ПВЗ: " + err.Error(),
		})
		return
	}

	// Формируем ответ с приёмками и товарами
	var response []models.PVZWithReceptionsResponse

	for _, pvz := range pvzList {
		// Получаем все приёмки для ПВЗ
		receptions, err := h.receptionQueries.GetReceptionsByPVZ(c.Request.Context(), pvz.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Message: "Ошибка при получении приёмок: " + err.Error(),
			})
			return
		}

		// Собираем информацию о приёмках и товарах
		receptionDetails := make([]models.ReceptionDetails, 0, len(receptions))

		for _, reception := range receptions {
			// Получаем товары для приёмки
			products, err := h.productQueries.GetProductsByReception(c.Request.Context(), reception.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorResponse{
					Message: "Ошибка при получении товаров: " + err.Error(),
				})
				return
			}

			// Преобразуем товары в ответ
			productResponses := make([]models.ProductResponse, 0, len(products))
			for _, product := range products {
				productResponses = append(productResponses, models.ProductResponse{
					ID:          product.ID,
					DateTime:    product.Datetime,
					Type:        product.Type,
					ReceptionID: product.ReceptionID,
				})
			}

			// Добавляем информацию о приёмке и товарах
			receptionDetails = append(receptionDetails, models.ReceptionDetails{
				Reception: models.ReceptionResponse{
					ID:       reception.ID,
					DateTime: reception.DateTime,
					PvzID:    reception.PvzID,
					Status:   reception.Status,
				},
				Products: productResponses,
			})
		}

		// Добавляем ПВЗ с приёмками в ответ
		response = append(response, models.PVZWithReceptionsResponse{
			PVZ: models.PVZResponse{
				ID:               pvz.ID,
				RegistrationDate: pvz.RegistrationDate,
				City:             pvz.City,
			},
			Receptions: receptionDetails,
		})
	}

	// Добавляем заголовок X-Total-Count для пагинации
	c.Header("X-Total-Count", fmt.Sprintf("%d", total))

	c.JSON(http.StatusOK, response)
}
