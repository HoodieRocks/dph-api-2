package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	nanoid "github.com/matoous/go-nanoid/v2"
)

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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
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
