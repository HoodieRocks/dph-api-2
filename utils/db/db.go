package db

import (
	"context"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/gommon/log"
)

var (
	pgInstance *postgres
	pgOnce     sync.Once
)

func EstablishConnection() *postgres {

	pgOnce.Do(func() {
		db, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))

		if err != nil {
			log.Errorf("unable to create connection pool: %v\n", err)
			os.Exit(1)
		}

		pgInstance = &postgres{db}
	})

	return pgInstance

}

func CreateTables(pg *postgres) {
	tx, err := pg.Db.Begin(context.Background())

	if err != nil {
		log.Errorf("failed to create tables: %v\n", err)
		os.Exit(1)
	}

	_, err = tx.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS users (
		id 			TEXT PRIMARY KEY,
		username 	VARCHAR(50) UNIQUE NOT NULL,
		role		VARCHAR(25) NOT NULL,
		bio			VARCHAR(2000) NOT NULL,
		badges		TEXT[],
		icon		TEXT,
		join_date	TIMESTAMP NOT NULL,
		password	VARCHAR(255) NOT NULL,
		token		TEXT NOT NULL UNIQUE
	)`)

	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback: %v\n", err)
			panic(err)
		}

		log.Errorf("failed to create user table: %v\n", err)
		panic(err)
	}

	_, err = tx.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS projects (
		id 				TEXT 			PRIMARY KEY,
		title 			VARCHAR(50) 	UNIQUE NOT NULL,
		slug			VARCHAR(50) 	UNIQUE NOT NULL,
		author			TEXT	 		NOT NULL REFERENCES users(id),
		description		VARCHAR(200) 	NOT NULL,
		body			VARCHAR(2000) 	NOT NULL,
		creation		TIMESTAMP 		NOT NULL,
		updated			TIMESTAMP 		NOT NULL,
		status			VARCHAR(255) 	NOT NULL DEFAULT 'draft',
		downloads		INTEGER			NOT NULL DEFAULT 0,
		category		TEXT[]			NOT NULL,
		icon			TEXT,
		license			TEXT,
		featured_until	TIMESTAMP
	)`)

	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback: %v\n", err)
			panic(err)
		}

		log.Errorf("failed to create project table: %v\n", err)
		panic(err)
	}

	_, err = tx.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS versions (
		id 				TEXT 			PRIMARY KEY,
		title 			VARCHAR(50) 	NOT NULL,
		description		VARCHAR(2000) 	NOT NULL,
		creation		TIMESTAMP 		NOT NULL,
		downloads		INTEGER			NOT NULL DEFAULT 0,
		download_link	TEXT			NOT NULL,
		version_code	TEXT			NOT NULL,
		supports		TEXT[]			NOT NULL,
		project			TEXT 		NOT NULL REFERENCES projects(id),
		rp_download		TEXT
	)`)

	if err != nil {
		newErr := tx.Rollback(context.Background())

		if newErr != nil {
			log.Errorf("failed to rollback: %v\n", err)
			panic(err)
		}

		log.Errorf("failed to create version table: %v\n", err)
		panic(err)
	}

	err = tx.Commit(context.Background())

	if err != nil {
		log.Errorf("failed to commit: %v\n", err.Error())
		panic(err)
	}
}
