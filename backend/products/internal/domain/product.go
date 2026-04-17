package domain

type Product struct {
	ID     int64  `json:"id"`
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	Price  int64  `json:"price"`
}

type SearchQuery struct {
	Name      string
	PriceFrom int64
	PriceTo   int64
}
