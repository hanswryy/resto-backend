package handlers

import (
	"firebase.google.com/go/v4/messaging"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	DB 			*pgxpool.Pool
	JWTSecret 	string
	FCM 		*messaging.Client
}

func New(db *pgxpool.Pool, jwtSecret string) *Handler {
	return &Handler{
		DB: db,
		JWTSecret: jwtSecret,
	}
}