CREATE TABLE notifications (
    id TEXT PRIMARY KEY NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    update_id TEXT NOT NULL,
    user_id TEXT REFERENCES users (id) ON DELETE CASCADE,
    level TEXT NOT NULL DEFAULT 'info',
    type TEXT NOT NULL DEFAULT 'info',
    data TEXT,
    title TEXT NOT NULL,
    description TEXT,
    read_at TIMESTAMP
);

CREATE INDEX notifications_user_id ON notifications (user_id);
CREATE INDEX notifications_update_id ON notifications (update_id);
