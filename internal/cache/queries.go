package cache

import (
	"database/sql"
	"fmt"

	"github.com/zyncc/ytmp/internal/models"
)

type Storer interface {
	UpsertPlaylists(playlists []models.Playlist) error
	FetchPlaylists(favoritesOnly bool) ([]Playlist, error)
	MarkFavoritePlaylist(id string) error
	UpsertSongs(playlistID string, songs []models.Song) error
	FetchAllSongs(playlistID string) ([]Song, error)
}

type CacheRepository struct {
	db *sql.DB
}

func NewCacheRepository(db *sql.DB) *CacheRepository {
	return &CacheRepository{
		db: db,
	}
}

func (c *CacheRepository) UpsertSongs(playlistID string, songs []models.Song) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO songs (
			id,
			playlist_id,
			title,
			url,
			duration,
			channel,
			thumbnail_url,
			view_count
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, playlist_id) DO UPDATE SET
			title = excluded.title,
			url = excluded.url,
			thumbnail_url = excluded.thumbnail_url,
			view_count = excluded.view_count,
			updated_at = CURRENT_TIMESTAMP
	`
	for _, song := range songs {
		if _, err := tx.Exec(
			query,
			song.ID,
			playlistID,
			song.Title,
			song.URL,
			song.Duration,
			song.Channel,
			song.Thumbnails[len(song.Thumbnails)-1].URL,
			song.ViewCount,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c *CacheRepository) FetchAllSongs(playlistID string) ([]Song, error) {
	rows, err := c.db.Query("SELECT id, playlist_id, title, url, duration, channel, thumbnail_url, view_count, created_at, updated_at FROM songs WHERE playlist_id = ?", playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []Song
	for rows.Next() {
		var song Song
		if err := rows.Scan(
			&song.ID,
			&song.PlaylistID,
			&song.Title,
			&song.URL,
			&song.Duration,
			&song.Channel,
			&song.ThumbnailURL,
			&song.ViewCount,
			&song.CreatedAt,
			&song.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := rows.Err(); err != nil {
			return nil, err
		}

		songs = append(songs, song)
	}

	return songs, nil
}

func (c *CacheRepository) MarkFavoritePlaylist(id string) error {
	result, err := c.db.Exec("UPDATE playlists SET favorite = NOT favorite, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("failed to find playlist with this id = %s", id)
	}

	return nil
}

func (c *CacheRepository) FetchPlaylists(favoritesOnly bool) ([]Playlist, error) {
	var query string

	if favoritesOnly {
		query = "SELECT id, title, url, thumbnail_url, created_at, updated_at FROM playlists WHERE favorite = 1 ORDER BY updated_at ASC"
	} else {
		query = "SELECT id, title, url, thumbnail_url, created_at, updated_at FROM playlists"
	}

	rows, err := c.db.Query(query)
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
