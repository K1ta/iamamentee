package models

import (
	"encoding/json"
	"errors"
)

const ProductEventTypeCreated = "created"

type Product struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Price  int64  `json:"price"`
}

func NewProduct(id int64, userID int64, name string, price int64) (*Product, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if name == "" {
		return nil, errors.New("empty name")
	}
	if price <= 0 {
		return nil, errors.New("invalid price")
	}
	return &Product{ID: id, UserID: userID, Name: name, Price: price}, nil
}

type ProductEvent struct {
	Type    string   `json:"type"`
	Product *Product `json:"product"`
}

func (e *ProductEvent) ToJSON() string {
	res, _ := json.Marshal(e)
	return string(res)
}
