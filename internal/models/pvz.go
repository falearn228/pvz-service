package models

import (
	"time"
)

// PVZ представляет пункт выдачи заказов
type PVZ struct {
	ID               string    `json:"id" db:"id"`
	RegistrationDate time.Time `json:"registrationDate" db:"registration_date"`
	City             string    `json:"city" db:"city"`
}

// CreatePVZRequest представляет запрос на создание ПВЗ
type CreatePVZRequest struct {
	City string `json:"city" binding:"required,oneof=Москва Санкт-Петербург Казань"`
}

// PVZResponse представляет ответ с данными ПВЗ
type PVZResponse struct {
	ID               string    `json:"id"`
	RegistrationDate time.Time `json:"registrationDate"`
	City             string    `json:"city"`
}

// PVZListQuery представляет параметры запроса для получения списка ПВЗ
type PVZListQuery struct {
	StartDate string `form:"startDate" time_format:"2006-01-02T15:04:05Z07:00"`
	EndDate   string `form:"endDate" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int    `form:"page" binding:"omitempty,min=1" default:"1"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=30" default:"10"`
}

// PVZWithReceptionsResponse представляет ответ со списком ПВЗ и связанными приёмками
type PVZWithReceptionsResponse struct {
	PVZ        PVZResponse        `json:"pvz"`
	Receptions []ReceptionDetails `json:"receptions"`
}

// ReceptionDetails представляет приёмку с товарами
type ReceptionDetails struct {
	Reception ReceptionResponse `json:"reception"`
	Products  []ProductResponse `json:"products"`
}
