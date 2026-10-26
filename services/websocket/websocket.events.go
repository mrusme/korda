package websocket

const (
	EventServerVersion = "on_server_version"
	EventUploadSuccess = "on_upload_success"
	EventUploadReady   = "AssetUploadReadyV2"
	EventEditReady     = "AssetEditReadyV2"
	EventAssetUpdate   = "on_asset_update"
	EventAssetHidden   = "on_asset_hidden"
	EventAssetTrash    = "on_asset_trash"
	EventAssetRestore  = "on_asset_restore"
	EventAssetDelete   = "on_asset_delete"
	EventStackUpdate   = "on_asset_stack_update"
	EventAlbumUpdate   = "on_album_update"
	EventNotification  = "on_notification"
	EventSessionDelete = "on_session_delete"
	EventUserDelete    = "on_user_delete"
)
