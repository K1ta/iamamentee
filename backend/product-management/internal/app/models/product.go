package models

import "errors"

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
	return &Product{UserID: userID, Name: name, Price: price}, nil
}
