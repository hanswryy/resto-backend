package models

import (
	"time"
)

type MenuItem struct {
	ID		  	int64	`json:"id"`
	Name		string	`json:"name"`
	Description	string	`json:"description"`
	Price		float64	`json:"price"`
	IsAvailable	bool	`json:"is_available"`
}

type Order struct {
	ID			int64		`json:"id"`
	UserID		int64		`json:"user_id"`
	Status		string		`json:"status"`
	Total 		int 		`json:"total"`
	CreatedAt	time.Time	`json:"created_at"`
	UpdatedAt	time.Time	`json:"updated_at"`
	Items		[]OrderItem	`json:"items"`
}

type OrderItem struct {
	ID				int64	`json:"id"`
	MenuItemID		int64	`json:"menu_item_id"`
	Name			string	`json:"name"` // JOIN from menu_items table
	Quantity		int		`json:"quantity"`
	PriceAtOrder	float64	`json:"price_at_order"`
}