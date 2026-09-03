package models

type MenuItem struct {
	ID		  	int64	`json:"id"`
	Name		string	`json:"name"`
	Description	string	`json:"description"`
	Price		float64	`json:"price"`
	IsAvailable	bool	`json:"is_available"`
}