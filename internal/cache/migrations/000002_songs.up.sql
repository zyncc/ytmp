CREATE TABLE IF NOT EXISTS songs (
    id TEXT,
    playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,

    title TEXT NOT NULL,
    url TEXT NOT NULL,
    duration INTEGER NOT NULL,
    channel TEXT,
    uploader TEXT,
    thumbnail_url TEXT,
    view_count INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY(id, playlist_id)
);