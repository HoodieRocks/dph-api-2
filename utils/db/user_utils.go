package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/gommon/log"
	nanoid "github.com/matoous/go-nanoid/v2"
)

func (pg *postgres) GetUserById(id string) (User, error) {
	var user User

	var row, err = pg.Db.Query(context.Background(), "SELECT * FROM users WHERE id = $1 LIMIT 1", id)

	if err != nil {
		return user, err
	}

	user, err = pgx.CollectOneRow(row, pgx.RowToStructByName[User])

	return user, err
}

func (pg *postgres) ResetUserToken(id string) error {
	var user, err = pg.GetUserById(id)

	if err != nil {
		return err
	}

	user.Token = generateSecureToken()

	tx, err := pg.Db.Begin(context.Background())

	if err != nil {
		err = tx.Rollback(context.Background())

		if err != nil {
			log.Errorf("failed to rollback: %v\n", err.Error())
			return err
		}

		return err
	}

	err = pg.UpdateUser(tx, user)

	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return err
	}

	return nil
}

func (pg *postgres) GetUserByToken(token string) (User, error) {
	var user User

	var row, err = pg.Db.Query(context.Background(), "SELECT * FROM users WHERE token = $1 LIMIT 1", token)

	if err != nil {
		return user, err
	}

	user, err = pgx.CollectOneRow(row, pgx.RowToStructByName[User])

	return user, err
}

func (pg *postgres) CheckForUsernameConflict(username string) bool {

	var rowLen = 0
	var err = pg.Db.QueryRow(context.Background(), `SELECT count(1) FROM users WHERE username = LOWER($1)`, username).Scan(&rowLen)

	return err == pgx.ErrNoRows || rowLen > 0
}

func (pg *postgres) CreateUser(tx pgx.Tx, user User) error {

	id, _ := nanoid.New(12)

	_, err := tx.Exec(context.Background(),
		"INSERT INTO users (id, username, role, bio, join_date, password, token) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		id,
		user.Username,
		user.Role,
		user.Bio,
		user.JoinDate,
		user.Password,
		user.Token)
	return err
}

func (pg *postgres) UpdateUser(tx pgx.Tx, user User) error {
	_, err := tx.Exec(context.Background(),
		`UPDATE users SET 
			username = $1, role = $2, bio = $3, join_date = $4, password = $5, token = $6 
		WHERE id = $7`,
		user.Username,
		user.Role,
		user.Bio,
		user.JoinDate,
		user.Password,
		user.Token,
		user.ID)
	return err
}

func (pg *postgres) GetAllInRole(id string) ([]User, error) {

	var users []User
	rows, err := pg.Db.Query(context.Background(), `SELECT * FROM users WHERE role = $1`, id)

	if err != nil {
		return nil, err
	}

	users, err = pgx.CollectRows(rows, pgx.RowToStructByName[User])

	if err != nil {
		return nil, err
	}

	return users, nil
}
