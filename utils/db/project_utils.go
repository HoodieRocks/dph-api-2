package db

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	nanoid "github.com/matoous/go-nanoid/v2"
)

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

func (pg *postgres) getProjectByX(column string, value string) (Project, error) {
	var project Project

	var row, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE `+column+` = $1 LIMIT 1`, value)

	if err != nil {
		return project, err
	}

	project, err = pgx.CollectOneRow(row, pgx.RowToStructByName[Project])

	return project, err
}

func (pg *postgres) GetProjectByID(id string) (Project, error) {
	return pg.getProjectByX("id", id)
}

func (pg *postgres) GetProjectBySlug(slug string) (Project, error) {
	return pg.getProjectByX("slug", slug)
}

func (pg *postgres) GetRandomProjects(limit int) ([]Project, error) {

	var project []Project
	var row, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE status = 'live' ORDER BY RANDOM() LIMIT $2`, limit)

	if err != nil {
		return project, err
	}

	project, err = pgx.CollectRows(row, pgx.RowToStructByName[Project])

	return project, err
}

func (pg *postgres) GetAllProjectsByAuthor(authorId string) ([]Project, error) {
	var project []Project

	var rows, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE author = $1`, authorId)

	if err != nil {
		return project, err
	}

	project, err = pgx.CollectRows(rows, pgx.RowToStructByName[Project])

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
	var err = pg.Db.QueryRow(context.Background(), "SELECT count(1) FROM projects WHERE title = LOWER($1) OR slug = LOWER($2)", title, slug).Scan(&rowLen)

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

func (pg *postgres) DeleteProject(tx pgx.Tx, projectId string) error {
	_, err := tx.Exec(context.Background(), `DELETE FROM projects WHERE id = $1 LIMIT 1`, projectId)
	return err
}

func (pg *postgres) FeatureProject(tx pgx.Tx, projectId string, featureUntil time.Duration) error {
	_, err := tx.Exec(context.Background(), `UPDATE projects SET featured_until = $1 WHERE id = $2`, time.Now().Add(featureUntil), projectId)
	return err
}

func (pg *postgres) GetFeaturedProjects() ([]Project, error) {
	var projects []Project
	var rows, err = pg.Db.Query(context.Background(), `SELECT `+PROJECT_COLUMNS+` FROM projects WHERE featured_until > $1`, time.Now())

	if err != nil {
		return nil, err
	}

	projects, err = pgx.CollectRows(rows, pgx.RowToStructByName[Project])

	if err != nil {
		return nil, err
	}

	return projects, nil
}
