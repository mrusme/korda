package errs

import (
	"errors"

	glideserrs "xn--gckvb8fzb.com/glides/errs"
)

var (
	ErrNoRows                  = glideserrs.ErrNoRows
	ErrUniqueViolationOn       = glideserrs.ErrUniqueViolationOn
	ErrUnavailable             = glideserrs.ErrUnavailable
	ErrDatabaseReadOnly        = glideserrs.ErrDatabaseReadOnly
	ErrHashInvalid             = glideserrs.ErrHashInvalid
	ErrHashVariantIncompatible = glideserrs.ErrHashVariantIncompatible
	ErrHashVersionIncompatible = glideserrs.ErrHashVersionIncompatible
	ErrJobTypeInvalid          = glideserrs.ErrJobTypeInvalid
	ErrJobSubTypeInvalid       = glideserrs.ErrJobSubTypeInvalid
	ErrJobPayloadInvalid       = glideserrs.ErrJobPayloadInvalid
)

var (
	ErrAuthenticationRequired = errors.New("Authentication required")
	ErrInvalidUserToken       = errors.New("Invalid user token")
	ErrInvalidAPIKey          = errors.New("Invalid API key")
	ErrInvalidShareKey        = errors.New("Invalid share key")
	ErrInvalidShareSlug       = errors.New("Invalid share slug")
	ErrIncorrectCredentials   = errors.New("Incorrect email or password")
	ErrUnauthorized           = errors.New("Unauthorized")
)

var (
	ErrForbidden              = errors.New("Forbidden")
	ErrMissingPermission      = errors.New("Missing required permission")
	ErrNotAPIKey              = errors.New("Not authenticated with an API Key")
	ErrCannotDeleteOwnAccount = errors.New("Cannot delete your own account")
)

var (
	ErrAdminSetupNotAvailable    = errors.New("Admin setup is not available")
	ErrEmailNotAvailable         = errors.New("Email is not available")
	ErrPasswordRequired          = errors.New("password is required")
	ErrFirstUserMustBeAdmin      = errors.New("The first registered account must the administrator.")
	ErrUserNotFound              = errors.New("User not found")
	ErrWrongPassword             = errors.New("Wrong password")
	ErrWrongPinCode              = errors.New("Wrong PIN code")
	ErrPinCodeExists             = errors.New("User already has a PIN code")
	ErrPinCodeMissing            = errors.New("User does not have a PIN code")
	ErrPasswordOrPinRequired     = errors.New("Either password or pinCode is required")
	ErrSessionTokenRequired      = errors.New("This endpoint can only be used with a session token")
	ErrNoFieldsToUpdate          = errors.New("No fields to update")
	ErrNoAccess                  = errors.New("Not found or no")
	ErrAPIKeyNotFound            = errors.New("API Key not found")
	ErrCannotGrantPermissions    = errors.New("Cannot grant permissions you do not have")
	ErrCannotRotateAPIKey        = errors.New("Cannot rotate an API Key with permissions you do not have")
	ErrAdminStatusSelf           = errors.New("Admin status can only be changed by another admin")
	ErrStorageLabelInUse         = errors.New("Storage label already in use by another account")
	ErrInvalidLicenseKey         = errors.New("Invalid license key")
	ErrProfileImageMissing       = errors.New("Can't delete a missing profile Image")
	ErrProfileImageUnprocessed   = errors.New("Unable to process profile image")
	ErrValidation                = errors.New("Validation failed")
	ErrBodyInvalid               = errors.New("Invalid request body")
	ErrQuotaExceeded             = errors.New("Quota has been exceeded!")
	ErrLivePhotoNotFound         = errors.New("Live photo video not found")
	ErrLivePhotoNotVideo         = errors.New("Live photo video must be a video")
	ErrLivePhotoNotOwned         = errors.New("Live photo video does not belong to the user")
	ErrFileRequired              = errors.New("File is required")
	ErrAssetDataEmpty            = errors.New("Field(s) assetData should not be empty")
	ErrAssetNotFound             = errors.New("Asset not found")
	ErrFieldTooLong              = errors.New("Field value too long")
	ErrUnexpectedEndOfForm       = errors.New("Multipart: Unexpected end of form")
	ErrUnlinkAndroidMotion       = errors.New("Cannot unlink Android motion photos")
	ErrElevatedRequired          = errors.New("Elevated permission is required")
	ErrBothAssetsMustExist       = errors.New("Both assets must exist")
	ErrSameSourceAndTarget       = errors.New("Source and target id must be distinct")
	ErrDuplicateItems            = errors.New("Duplicate items are not allowed")
	ErrMetadataNotFound          = errors.New("Metadata with key")
	ErrSyncAPIKey                = errors.New("Sync endpoints cannot be used with API keys")
	ErrSyncTypeDeprecated        = errors.New("SyncRequestType.")
	ErrInvalidAckType            = errors.New("Invalid ack type:")
	ErrRealtimeTranscoding       = errors.New("Real-time transcoding is not enabled")
	ErrSmartSearchDisabled       = errors.New("Smart search is not enabled")
	ErrInvalidCursor             = errors.New("Invalid cursor")
	ErrInvalidUser               = errors.New("Invalid user")
	ErrPartnerExists             = errors.New("Partner already exists")
	ErrPartnerNotFound           = errors.New("Partner not found")
	ErrStackNotFound             = errors.New("Asset stack not found")
	ErrPrimaryNotInStack         = errors.New("Primary asset must be in the stack")
	ErrAssetNotInStack           = errors.New("Asset not in stack")
	ErrRemovePrimaryAsset        = errors.New("Cannot remove stack's primary asset")
	ErrTagNotFound               = errors.New("Tag not found")
	ErrTagExists                 = errors.New("A tag with that name already exists")
	ErrAlbumNotFound             = errors.New("Album not found")
	ErrInvalidAlbumThumbnail     = errors.New("Invalid album thumbnail")
	ErrAnotherOwner              = errors.New("Cannot add another owner")
	ErrUserIsOwner               = errors.New("User is owner")
	ErrAlbumNotShared            = errors.New("Album not shared with user")
	ErrLastAlbumOwner            = errors.New("Cannot remove the last album owner")
	ErrSharedLinkSearch          = errors.New("Shared link access is only allowed in combination with an albumIds filter")
	ErrMemoryNotFound            = errors.New("Memory not found")
	ErrSharedLinkNotFound        = errors.New("Shared link not found")
	ErrSharedLinkSave            = errors.New("Failed to save shared link")
	ErrSharedLinkType            = errors.New("Invalid shared link type")
	ErrSharedLinkNoPassword      = errors.New("Shared link is not password protected")
	ErrSharedLinkPassword        = errors.New("Invalid password")
	ErrSharedLinkPasswordMissing = errors.New("Password required")
	ErrNotificationNotFound      = errors.New("Notification not found")
	ErrSMTPVerify                = errors.New("Failed to verify SMTP configuration")
	ErrQueueNotPausable          = errors.New("The BackgroundTask queue cannot be paused")
	ErrNotInMaintenance          = errors.New("Not in maintenance mode")
	ErrInvalidBackupName         = errors.New("Invalid backup name!")
	ErrOwnClusterGroup           = errors.New("Cannot request to join your own cluster group")
	ErrLastClusterMember         = errors.New("Cannot leave a cluster group without any other members")
	ErrPluginNotFound            = errors.New("Plugin not found")
	ErrPluginTypeFilter          = errors.New("err_plugin_type_filter")
	ErrOAuthDisabled             = errors.New("OAuth is not enabled")
	ErrOAuthState                = errors.New("OAuth state is missing")
	ErrOAuthVerifier             = errors.New("OAuth code verifier is missing")
	ErrOAuthLogout               = errors.New("Received backchannel logout request but OAuth is not enabled")
	ErrConfigFile                = errors.New("Cannot update configuration while IMMICH_CONFIG_FILE is in use")
	ErrLibraryNotFound           = errors.New("Library not found")
	ErrPersonNotFound            = errors.New("Person not found")
	ErrPersonOtherUser           = errors.New("Only name and birthDate can be updated for other users")
	ErrMergeTooFew               = errors.New("At least two people are required for merging")
	ErrMergeSelf                 = errors.New("Cannot merge a person into themselves")
	ErrShareWithSelf             = errors.New("Cannot share a person with yourself")
	ErrClusterGroup              = errors.New("All users must be in the same cluster group")
	ErrSidecarNotDeletable       = errors.New("Sidecar files cannot be deleted")
	ErrAssetFileNotFound         = errors.New("Asset file not found")
	ErrOnlyImagesEditable        = errors.New("Only images can be edited")
	ErrEditLivePhoto             = errors.New("Editing live photos is not supported")
	ErrEditPanorama              = errors.New("Editing panorama images is not supported")
	ErrEditGIF                   = errors.New("Editing GIF images is not supported")
	ErrEditSVG                   = errors.New("Editing SVG images is not supported")
	ErrEditDimensions            = errors.New("Asset dimensions are not available for editing")
	ErrCropFirst                 = errors.New("Crop action must be the first edit action")
	ErrCropBounds                = errors.New("Crop parameters are out of bounds")
	ErrDownloadTarget            = errors.New("assetIds, albumId, or userId is required")
	ErrOtherLockedTimeline       = errors.New("You may not access another user's locked timeline")
	ErrTimeBucketFormat          = errors.New("err_time_bucket_format")
	ErrWithPartners              = errors.New("withPartners is only supported for non-archived, non-trashed, non-favorited, non-locked assets")
	ErrJobRunning                = errors.New("Job is already running")
	ErrInvalidJobName            = errors.New("Invalid job name")
)

type UnknownMethod struct {
	Method string
}

func (e *UnknownMethod) Error() string {
	return "Unknown method " + e.Method
}

type Unsupported string

func (u Unsupported) Error() string {
	return string(u)
}

const (
	UnsupportedBackups     Unsupported = "Database backups are not supported by korda"
	UnsupportedIntegrity   Unsupported = "Integrity reports are not supported by korda"
	UnsupportedLibraries   Unsupported = "External libraries are not supported by korda"
	UnsupportedPeople      Unsupported = "People and faces are not supported by korda"
	UnsupportedWorkflows   Unsupported = "Workflows are not supported by korda"
	UnsupportedPlugins     Unsupported = "Plugins are not supported by korda"
	UnsupportedMaintenance Unsupported = "Maintenance mode is not supported by korda"
	UnsupportedClusters    Unsupported = "Cluster group invitations are not supported by korda"
)

var (
	ErrNotFound            = errors.New("Not Found")
	ErrAssetMediaNotFound  = errors.New("Asset media not found")
	ErrStreamNotReady      = errors.New("Asset not found or metadata not yet ready for streaming")
	ErrTooManyRequests     = errors.New("Too Many Requests")
	ErrPayloadTooLarge     = errors.New("Payload Too Large")
	ErrInternal            = errors.New("Internal server error")
	ErrUnsupportedFileType = errors.New("Unsupported file type")
	ErrFileEmpty           = errors.New("File is empty")
	ErrUnexpectedFileField = errors.New("Unexpected file field")
)

var (
	ErrMediaRootInvalid  = errors.New("err_media_root_invalid")
	ErrImageUndecodable  = errors.New("err_image_undecodable")
	ErrServiceMissing    = errors.New("err_service_missing")
	ErrCreateUserInvalid = errors.New("err_create_user_invalid")
	ErrJobsTargetMissing = errors.New("err_jobs_target_missing")
)
