CREATE TABLE users (
    id TEXT PRIMARY KEY NOT NULL,
    email TEXT NOT NULL,
    password TEXT,
    pin_code TEXT,
    name TEXT NOT NULL DEFAULT '',
    is_admin INTEGER NOT NULL DEFAULT 0,
    should_change_password INTEGER NOT NULL DEFAULT 1,
    avatar_color TEXT,
    profile_image_path TEXT NOT NULL DEFAULT '',
    profile_changed_at TIMESTAMP NOT NULL,
    storage_label TEXT,
    quota_size_in_bytes INTEGER,
    quota_usage_in_bytes INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    oauth_id TEXT,
    cluster_group_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX users_email ON users (email);
CREATE UNIQUE INDEX users_storage_label ON users (storage_label);
CREATE INDEX users_update_id ON users (update_id);
CREATE INDEX users_deleted_at ON users (deleted_at);

CREATE TABLE user_audit (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX user_audit_deleted_at ON user_audit (deleted_at);

CREATE TABLE user_metadata (
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (user_id, key)
);

CREATE INDEX user_metadata_update_id ON user_metadata (update_id);

CREATE TABLE user_metadata_audit (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL,
    key TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX user_metadata_audit_user_id ON user_metadata_audit (user_id);
CREATE INDEX user_metadata_audit_deleted_at ON user_metadata_audit (deleted_at);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY NOT NULL,
    token BLOB NOT NULL,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES sessions (id) ON DELETE CASCADE,
    device_type TEXT NOT NULL DEFAULT '',
    device_os TEXT NOT NULL DEFAULT '',
    app_version TEXT,
    expires_at TIMESTAMP,
    pin_expires_at TIMESTAMP,
    is_pending_sync_reset INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX sessions_token ON sessions (token);
CREATE INDEX sessions_user_id ON sessions (user_id);
CREATE INDEX sessions_parent_id ON sessions (parent_id);

CREATE TABLE sync_checkpoints (
    session_id TEXT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    ack TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (session_id, type)
);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY NOT NULL,
    name TEXT NOT NULL,
    key BLOB NOT NULL,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permissions TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX api_keys_key ON api_keys (key);
CREATE INDEX api_keys_user_id ON api_keys (user_id);

CREATE TABLE system_metadata (
    key TEXT PRIMARY KEY NOT NULL,
    value TEXT NOT NULL
);

CREATE TABLE version_history (
    id TEXT PRIMARY KEY NOT NULL,
    created_at TIMESTAMP NOT NULL,
    version TEXT NOT NULL
);
