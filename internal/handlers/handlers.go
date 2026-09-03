package handlers

import "github.com/jackc/pgx/v5/pgxpool"

type Handler struct {
	DB *pgxpool.Pool
	JWTSecret string
}

func New(db *pgxpool.Pool, jwtSecret string) *Handler {
	return &Handler{
		DB: db,
		JWTSecret: jwtSecret,
	}
}