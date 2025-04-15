package models

import "time"

// Reception представляет приёмку товаров
type Reception struct {
	ID       string    `json:"id" db:"id"`
	DateTime time.Time `json:"dateTime" db:"datetime"`
	PvzID    string    `json:"pvzId" db:"pvz_id"`
	Status   string    `json:"status" db:"status"`
}

// CreateReceptionRequest представляет запрос на создание приёмки товаров
type CreateReceptionRequest struct {
	PvzID string `json:"pvzId" binding:"required,uuid"`
}

// ReceptionResponse представляет ответ с данными приёмки
type ReceptionResponse struct {
	ID       string    `json:"id"`
	DateTime time.Time `json:"dateTime"`
	PvzID    string    `json:"pvzId"`
	Status   string    `json:"status"`
}
