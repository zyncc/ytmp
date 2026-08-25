package cache

import (
	"database/sql"

	"github.com/zyncc/ytmp/internal/models"
)

type Storer interface {
	UpsertPlaylists(playlists []models.Playlist) error
	FetchPlaylists() ([]Playlist, error)
}

type CacheRepository struct {
	db *sql.DB
}

func NewCacheRepository(db *sql.DB) *CacheRepository {
	return &CacheRepository{
		db: db,
	}
}

func (c *CacheRepository) FetchPlaylists() ([]Playlist, error) {
	rows, err := c.db.Query("SELECT id, title, url, thumbnail_url, created_at, updated_at FROM playlists")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []Playlist

	for rows.Next() {
		var p Playlist

		err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.URL,
			&p.ThumbnailURL,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}

		playlists = append(playlists, p)
	}

	return playlists, nil
}

func (c *CacheRepository) UpsertPlaylists(playlists []models.Playlist) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO playlists (
			id,
			title,
			url,
			thumbnail_url
		)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			url = excluded.url,
			thumbnail_url = excluded.thumbnail_url,
			updated_at = CURRENT_TIMESTAMP
	`
	for _, playlist := range playlists {
		if _, err := tx.Exec(
			query,
			playlist.ID,
			playlist.Title,
			playlist.URL,
			playlist.Thumbnails[len(playlist.Thumbnails)-1].URL,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
