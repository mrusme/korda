CREATE TABLE assets (
    id TEXT PRIMARY KEY NOT NULL,
    owner_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    original_path TEXT NOT NULL,
    original_file_name TEXT NOT NULL,
    checksum BLOB NOT NULL,
    checksum_algorithm TEXT NOT NULL DEFAULT 'sha1',
    file_created_at TIMESTAMP NOT NULL,
    file_modified_at TIMESTAMP NOT NULL,
    local_date_time TIMESTAMP NOT NULL,
    is_favorite INTEGER NOT NULL DEFAULT 0,
    duration INTEGER,
    live_photo_video_id TEXT REFERENCES assets (id) ON DELETE SET NULL,
    thumbhash BLOB,
    visibility TEXT NOT NULL DEFAULT 'timeline',
    status TEXT NOT NULL DEFAULT 'active',
    deleted_at TIMESTAMP,
    stack_id TEXT,
    library_id TEXT,
    is_offline INTEGER NOT NULL DEFAULT 0,
    is_external INTEGER NOT NULL DEFAULT 0,
    is_edited INTEGER NOT NULL DEFAULT 0,
    duplicate_id TEXT,
    width INTEGER,
    height INTEGER,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX assets_owner_checksum ON assets (owner_id, checksum) WHERE library_id IS NULL;
CREATE INDEX assets_checksum ON assets (checksum);
CREATE INDEX assets_owner_local_date_time ON assets (owner_id, local_date_time);
CREATE INDEX assets_owner_file_created_at ON assets (owner_id, file_created_at);
CREATE INDEX assets_owner_update_id ON assets (owner_id, update_id);
CREATE INDEX assets_update_id ON assets (update_id);
CREATE INDEX assets_stack_id ON assets (stack_id);
CREATE INDEX assets_live_photo_video_id ON assets (live_photo_video_id);
CREATE INDEX assets_deleted_at ON assets (deleted_at);
CREATE INDEX assets_original_path ON assets (original_path);
CREATE INDEX assets_original_file_name ON assets (original_file_name);
CREATE INDEX assets_duplicate_id ON assets (duplicate_id);
CREATE INDEX assets_created_at ON assets (created_at);

CREATE TABLE asset_audit (
    id TEXT PRIMARY KEY NOT NULL,
    asset_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX asset_audit_asset_id ON asset_audit (asset_id);
CREATE INDEX asset_audit_owner_id ON asset_audit (owner_id);
CREATE INDEX asset_audit_deleted_at ON asset_audit (deleted_at);

CREATE TABLE asset_exif (
    asset_id TEXT PRIMARY KEY NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    make TEXT,
    model TEXT,
    exif_image_width INTEGER,
    exif_image_height INTEGER,
    file_size_in_byte INTEGER,
    orientation TEXT,
    date_time_original TIMESTAMP,
    modify_date TIMESTAMP,
    lens_model TEXT,
    f_number REAL,
    focal_length REAL,
    iso INTEGER,
    latitude REAL,
    longitude REAL,
    city TEXT,
    state TEXT,
    country TEXT,
    description TEXT NOT NULL DEFAULT '',
    fps REAL,
    exposure_time TEXT,
    live_photo_cid TEXT,
    time_zone TEXT,
    projection_type TEXT,
    profile_description TEXT,
    colorspace TEXT,
    bits_per_sample INTEGER,
    auto_stack_id TEXT,
    rating INTEGER,
    tags TEXT,
    locked_properties TEXT,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE INDEX asset_exif_city ON asset_exif (city);
CREATE INDEX asset_exif_make ON asset_exif (make);
CREATE INDEX asset_exif_model ON asset_exif (model);
CREATE INDEX asset_exif_live_photo_cid ON asset_exif (live_photo_cid);
CREATE INDEX asset_exif_auto_stack_id ON asset_exif (auto_stack_id);
CREATE INDEX asset_exif_update_id ON asset_exif (update_id);

CREATE TABLE asset_files (
    id TEXT PRIMARY KEY NOT NULL,
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    path TEXT NOT NULL,
    is_edited INTEGER NOT NULL DEFAULT 0,
    is_progressive INTEGER NOT NULL DEFAULT 0,
    is_transparent INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL
);

CREATE UNIQUE INDEX asset_files_asset_type ON asset_files (asset_id, type, is_edited);
CREATE INDEX asset_files_update_id ON asset_files (update_id);

CREATE TABLE asset_metadata (
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    PRIMARY KEY (asset_id, key)
);

CREATE INDEX asset_metadata_update_id ON asset_metadata (update_id);
CREATE INDEX asset_metadata_updated_at ON asset_metadata (updated_at);

CREATE TABLE asset_metadata_audit (
    id TEXT PRIMARY KEY NOT NULL,
    asset_id TEXT NOT NULL,
    key TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX asset_metadata_audit_asset_id ON asset_metadata_audit (asset_id);
CREATE INDEX asset_metadata_audit_key ON asset_metadata_audit (key);
CREATE INDEX asset_metadata_audit_deleted_at ON asset_metadata_audit (deleted_at);

CREATE TABLE asset_job_status (
    asset_id TEXT PRIMARY KEY NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    metadata_extracted_at TIMESTAMP,
    thumbnails_generated_at TIMESTAMP
);
