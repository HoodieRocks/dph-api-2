package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgres struct {
	Db *pgxpool.Pool
}

// ! MISC

func (pg *postgres) Ping() error {
	return pg.Db.Ping(context.Background())
}
