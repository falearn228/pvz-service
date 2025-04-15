package models

import (
	"time"
)

// Типы товаров
const (
	ProductTypeElectronics = "электроника"
	ProductTypeClothes     = "одежда"
	ProductTypeShoes       = "обувь"
)

// Product представляет товар
type Product struct {
	ID                string    `json:"id" db:"id"`
	ReceptionDatetime time.Time `json:"dateTime" db:"datetime"`
	Type              string    `json:"type" db:"type"`
	ReceptionID       string    `json:"receptionId" db:"reception_id"`
}

// CreateProductRequest представляет запрос на добавление товара
type CreateProductRequest struct {
	Type  string `json:"type" binding:"required,oneof=электроника одежда обувь"`
	PvzID string `json:"pvzId" binding:"required,uuid"`
}

// ProductResponse представляет ответ с данными товара
type ProductResponse struct {
	ID          string    `json:"id"`
	DateTime    time.Time `json:"dateTime"`
	Type        string    `json:"type"`
	ReceptionID string    `json:"receptionId"`
}
