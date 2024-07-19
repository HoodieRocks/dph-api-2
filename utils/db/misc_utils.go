package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	Db *pgxpool.Pool
}

// ! MISC

func (pg *postgres) Ping() error {
	return pg.Db.Ping(context.Background())
}

func generateSecureToken() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
