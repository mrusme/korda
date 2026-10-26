CREATE TABLE asset_edits (
    id TEXT PRIMARY KEY NOT NULL,
    asset_id TEXT NOT NULL REFERENCES assets (id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    parameters TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    update_id TEXT NOT NULL,
    UNIQUE (asset_id, sequence)
);

CREATE INDEX asset_edits_update_id ON asset_edits (update_id);

CREATE TABLE asset_edit_audit (
    id TEXT PRIMARY KEY NOT NULL,
    edit_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    deleted_at TIMESTAMP NOT NULL
);

CREATE INDEX asset_edit_audit_asset_id ON asset_edit_audit (asset_id);
CREATE INDEX asset_edit_audit_deleted_at ON asset_edit_audit (deleted_at);
