CREATE TABLE albums (
    id TEXT PRIMARY KEY NOT NULL,
    album_name TEXT NOT NULL DEFAULT 'Untitled Album',
    description TEXT,
    album_thumbnail_asset_id TEXT REFERENCES assets (id) ON DELETE SET NULL,
    is_activity_enabled INTEGER NOT NULL DEFAULT 1,
    "order" TEXT NOT NULL DEFAULT 'desc',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    update_id TEXT NOT NULL
);

CREATE INDEX albums_album_thumbnail_asset_id ON albums (album_thumbnail_asset_id);
CREATE INDEX albums_update_id ON albums (update_id);

CREATE TABLE album_users (
    album_id TEXT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'editor',
    create_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (album_id, user_id)
);

CREATE UNIQUE INDEX album_users_owner ON album_users (album_id) WHERE role = 'owner';
CREATE INDEX album_users_user_id_create_id ON album_users (user_id, create_id);
CREATE INDEX album_users_update_id ON album_users (update_id);

CREATE TABLE album_assets (
    album_id TEXT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (album_id, asset_id)
);

CREATE INDEX album_assets_asset_id ON album_assets (asset_id);
CREATE INDEX album_assets_update_id ON album_assets (update_id);

CREATE TABLE album_audit (
    id TEXT PRIMARY KEY NOT NULL,
    album_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX album_audit_album_id ON album_audit (album_id);
CREATE INDEX album_audit_user_id ON album_audit (user_id);
CREATE INDEX album_audit_deleted_at ON album_audit (deleted_at);

CREATE TABLE album_user_audit (
    id TEXT PRIMARY KEY NOT NULL,
    album_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX album_user_audit_album_id ON album_user_audit (album_id);
CREATE INDEX album_user_audit_user_id ON album_user_audit (user_id);
CREATE INDEX album_user_audit_deleted_at ON album_user_audit (deleted_at);

CREATE TABLE album_asset_audit (
    id TEXT PRIMARY KEY NOT NULL,
    album_id TEXT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX album_asset_audit_album_id ON album_asset_audit (album_id);
CREATE INDEX album_asset_audit_asset_id ON album_asset_audit (asset_id);
CREATE INDEX album_asset_audit_deleted_at ON album_asset_audit (deleted_at);

CREATE TABLE partners (
    shared_by_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    shared_with_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    in_timeline INTEGER NOT NULL DEFAULT 0,
    create_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (shared_by_id, shared_with_id)
);

CREATE INDEX partners_shared_with_id_create_id ON partners (shared_with_id, create_id);
CREATE INDEX partners_update_id ON partners (update_id);

CREATE TABLE partner_audit (
    id TEXT PRIMARY KEY NOT NULL,
    shared_by_id TEXT NOT NULL,
    shared_with_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX partner_audit_shared_by_id ON partner_audit (shared_by_id);
CREATE INDEX partner_audit_shared_with_id ON partner_audit (shared_with_id);
CREATE INDEX partner_audit_deleted_at ON partner_audit (deleted_at);

CREATE TABLE stacks (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    primary_asset_id TEXT NOT NULL REFERENCES assets (id),
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX stacks_primary_asset_id ON stacks (primary_asset_id);
CREATE INDEX stacks_owner_id ON stacks (owner_id);
CREATE INDEX stacks_update_id ON stacks (update_id);

CREATE TABLE stack_audit (
    id TEXT PRIMARY KEY NOT NULL,
    stack_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX stack_audit_user_id ON stack_audit (user_id);
CREATE INDEX stack_audit_deleted_at ON stack_audit (deleted_at);

CREATE TABLE tags (
    id TEXT PRIMARY KEY NOT NULL,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    value TEXT NOT NULL,
    color TEXT,
    parent_id TEXT REFERENCES tags (id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX tags_user_id_value ON tags (user_id, value);
CREATE INDEX tags_parent_id ON tags (parent_id);
CREATE INDEX tags_update_id ON tags (update_id);

CREATE TABLE tag_assets (
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (asset_id, tag_id)
);

CREATE INDEX tag_assets_tag_id ON tag_assets (tag_id);

CREATE TABLE shared_links (
    id TEXT PRIMARY KEY NOT NULL,
    description TEXT,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    key BLOB NOT NULL,
    type TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP,
    allow_upload INTEGER NOT NULL DEFAULT 0,
    album_id TEXT REFERENCES albums (id) ON DELETE CASCADE,
    allow_download INTEGER NOT NULL DEFAULT 1,
    show_exif INTEGER NOT NULL DEFAULT 1,
    password TEXT,
    slug TEXT
);

CREATE UNIQUE INDEX shared_links_key ON shared_links (key);
CREATE UNIQUE INDEX shared_links_slug ON shared_links (slug);
CREATE INDEX shared_links_user_id ON shared_links (user_id);
CREATE INDEX shared_links_album_id ON shared_links (album_id);

CREATE TABLE shared_link_assets (
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    shared_link_id TEXT NOT NULL REFERENCES shared_links (id) ON DELETE CASCADE,
    PRIMARY KEY (asset_id, shared_link_id)
);

CREATE INDEX shared_link_assets_shared_link_id ON shared_link_assets (shared_link_id);

CREATE TABLE activities (
    id TEXT PRIMARY KEY NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    album_id TEXT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    asset_id TEXT REFERENCES assets (id) ON DELETE CASCADE,
    comment TEXT,
    is_liked INTEGER NOT NULL DEFAULT 0,
    update_id TEXT NOT NULL,
    FOREIGN KEY (album_id, asset_id) REFERENCES album_assets (album_id, asset_id) ON DELETE CASCADE,
    CHECK ((comment IS NULL AND is_liked = 1) OR (comment IS NOT NULL AND is_liked = 0))
);

CREATE UNIQUE INDEX activities_like ON activities (asset_id, user_id, album_id) WHERE is_liked = 1;
CREATE INDEX activities_album_id ON activities (album_id);
CREATE INDEX activities_user_id ON activities (user_id);
CREATE INDEX activities_asset_id ON activities (asset_id);
CREATE INDEX activities_update_id ON activities (update_id);

CREATE TABLE memories (
    id TEXT PRIMARY KEY NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP,
    owner_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    data TEXT NOT NULL,
    is_saved INTEGER NOT NULL DEFAULT 0,
    memory_at TIMESTAMP NOT NULL,
    seen_at TIMESTAMP,
    show_at TIMESTAMP,
    hide_at TIMESTAMP,
    update_id TEXT NOT NULL
);

CREATE INDEX memories_owner_id ON memories (owner_id);
CREATE INDEX memories_update_id ON memories (update_id);

CREATE TABLE memory_assets (
    memory_id TEXT NOT NULL REFERENCES memories (id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (memory_id, asset_id)
);

CREATE INDEX memory_assets_asset_id ON memory_assets (asset_id);
CREATE INDEX memory_assets_update_id ON memory_assets (update_id);

CREATE TABLE memory_audit (
    id TEXT PRIMARY KEY NOT NULL,
    memory_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX memory_audit_memory_id ON memory_audit (memory_id);
CREATE INDEX memory_audit_user_id ON memory_audit (user_id);
CREATE INDEX memory_audit_deleted_at ON memory_audit (deleted_at);

CREATE TABLE memory_asset_audit (
    id TEXT PRIMARY KEY NOT NULL,
    memory_id TEXT NOT NULL REFERENCES memories (id) ON DELETE CASCADE,
    asset_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX memory_asset_audit_memory_id ON memory_asset_audit (memory_id);
CREATE INDEX memory_asset_audit_asset_id ON memory_asset_audit (asset_id);
CREATE INDEX memory_asset_audit_deleted_at ON memory_asset_audit (deleted_at);
