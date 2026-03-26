package app

type Product struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Price  int64  `json:"price"`
}
