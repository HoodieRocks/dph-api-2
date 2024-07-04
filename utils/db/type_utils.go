package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	nanoid "github.com/matoous/go-nanoid/v2"
)

type postgres struct {
	Db *pgxpool.Pool
}

var PROJECT_COLUMNS = `id,
			title,
			slug,
			author,
			description,
			body,
			creation,
			updated,
			status,
			downloads,
			category,
			icon,
			license,
			featured_until`

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
			fmt.Fprintf(os.Stderr, "failed to rollback: %v\n", err.Error())
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
	var err = pg.Db.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE username = LOWER($1)`, username).Scan(&rowLen)

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

// ! PROJECTS

func (pg *postgres) ListProjects(limit int, offset int, searchMethod string) ([]Project, error) {

	var trueLimit = limit
	if trueLimit > 100 {
		trueLimit = 100
	}

	rows, err := pg.Db.Query(context.Background(),
		`SELECT `+PROJECT_COLUMNS+`
			FROM projects
			WHERE status = 'live'
			ORDER BY $1 ASC
			LIMIT $2 OFFSET $3`,
		searchMethod,
		trueLimit,
		offset)

	if err != nil {

		if err == pgx.ErrNoRows {
			return []Project{}, nil
		}

		return nil, err
	}

	projects, err := pgx.CollectRows(rows, pgx.RowToStructByName[Project])

	return projects, err
}

func (pg *postgres) GetProjectByID(id string) (Project, error) {
	var project Project

	var row, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE id = $1 LIMIT 1`, id)

	if err != nil {
		return project, err
	}

	project, err = pgx.CollectOneRow(row, pgx.RowToStructByName[Project])

	return project, err
}

func (pg *postgres) GetProjectBySlug(slug string) (Project, error) {
	var project Project

	var row, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE slug = $1 LIMIT 1`, slug)

	if err != nil {
		return project, err
	}

	project, err = pgx.CollectOneRow(row, pgx.RowToStructByName[Project])

	return project, err
}

func (pg *postgres) FTSSearchProjects(query string) (pgx.Rows, error) {
	var rows, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE fts_column @@ to_tsquery('english',$1) AND status = 'live' LIMIT 100`, query)
	return rows, err
}

func (pg *postgres) SearchProjects(query string) (pgx.Rows, error) {
	var rows, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE 
		(
			title LIKE $1 OR 
			description LIKE $1 OR 
			slug LIKE $1
		) AND status = 'live' LIMIT 100`, "%"+query+"%")
	return rows, err
}

func (pg *postgres) CheckForProjectNameConflict(title string, slug string) bool {

	var rowLen = 0
	var err = pg.Db.QueryRow(context.Background(), "SELECT count(*) FROM projects WHERE title = LOWER($1) OR slug = LOWER($2)", title, slug).Scan(&rowLen)

	return err == pgx.ErrNoRows || rowLen > 0
}

func (pg *postgres) CreateProject(tx pgx.Tx, project Project) error {

	id, _ := nanoid.New(12)

	_, err := tx.Exec(context.Background(),
		"INSERT INTO projects (id, title, slug, author, description, body, creation, updated, category) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		id,
		project.Title,
		project.Slug,
		project.Author,
		project.Description,
		project.Body,
		project.Creation,
		project.Updated,
		project.Category)
	return err
}

func (pg *postgres) UpdateProject(tx pgx.Tx, project Project) error {
	query := `UPDATE projects SET 
		author = $1, 
		body = $2, 
		category = $3, 
		creation = $4, 
		description = $5, 
		downloads = $6, 
		featured_until = $7,
		icon = $8,
		license = $9,
		slug = $10,
		status = $11,
		title = $12,
		updated = $13
	WHERE id = $14`

	params := []interface{}{
		project.Author,
		project.Body,
		project.Category,
		project.Creation,
		project.Description,
		project.Downloads,
		project.FeaturedUntil,
		project.Icon,
		project.License,
		project.Slug,
		project.Status,
		project.Title,
		project.Updated,
		project.ID,
	}

	_, err := tx.Exec(context.Background(), query, params...)
	return err
}

func (pg *postgres) UpdateProjectDownloads(tx pgx.Tx, projectId string, downloads int) error {
	_, err := tx.Exec(context.Background(), `UPDATE projects SET downloads = $1 WHERE id = $2`, downloads, projectId)
	return err
}

func (pg *postgres) UpdateProjectStatus(tx pgx.Tx, projectId string, status string) error {
	_, err := tx.Exec(context.Background(), `UPDATE projects SET status = $1 WHERE id = $2`, strings.ToLower(status), projectId)
	return err
}

func (pg *postgres) CreateVersion(tx pgx.Tx, projectId string, version Version) error {

	id, _ := nanoid.New(12)

	_, err := tx.Exec(context.Background(),
		`INSERT INTO versions (
			id,
			title,
			description,
			creation,
			downloads,
			download_link,
			version_code,
			supports,
			project,
			rp_download) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id,
		version.Title,
		version.Description,
		version.Creation,
		0,
		version.DownloadLink,
		version.VersionCode,
		version.Supports,
		projectId,
		version.RpDownload)
	return err
}

// ! VERSIONS

func (pg *postgres) GetAllProjectVersions(projectId string) ([]Version, error) {
	var versions []Version

	var row, err = pg.Db.Query(context.Background(), `SELECT * FROM versions WHERE project = $1`, projectId)

	if err != nil {
		return versions, err
	}

	versions, err = pgx.CollectRows(row, pgx.RowToStructByName[Version])

	return versions, err
}

func (pg *postgres) GetVersionByCreation(projectId string, idx int) (*Version, error) {
	var row, err = pg.Db.Query(context.Background(), `SELECT * FROM versions WHERE project = $1 ORDER BY creation`, projectId)

	if err != nil {
		return nil, err
	}

	versions, err := pgx.CollectRows(row, pgx.RowToStructByName[Version])

	if err != nil || int(idx) > len(versions) {
		return nil, err
	}

	return &versions[idx], err
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
