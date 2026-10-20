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
	ErrAdminSetupNotAvailable  = errors.New("Admin setup is not available")
	ErrEmailNotAvailable       = errors.New("Email is not available")
	ErrPasswordRequired        = errors.New("password is required")
	ErrFirstUserMustBeAdmin    = errors.New("The first registered account must the administrator.")
	ErrUserNotFound            = errors.New("User not found")
	ErrWrongPassword           = errors.New("Wrong password")
	ErrWrongPinCode            = errors.New("Wrong PIN code")
	ErrPinCodeExists           = errors.New("User already has a PIN code")
	ErrPinCodeMissing          = errors.New("User does not have a PIN code")
	ErrPasswordOrPinRequired   = errors.New("Either password or pinCode is required")
	ErrSessionTokenRequired    = errors.New("This endpoint can only be used with a session token")
	ErrNoFieldsToUpdate        = errors.New("No fields to update")
	ErrNoAccess                = errors.New("Not found or no")
	ErrAPIKeyNotFound          = errors.New("API Key not found")
	ErrCannotGrantPermissions  = errors.New("Cannot grant permissions you do not have")
	ErrCannotRotateAPIKey      = errors.New("Cannot rotate an API Key with permissions you do not have")
	ErrAdminStatusSelf         = errors.New("Admin status can only be changed by another admin")
	ErrStorageLabelInUse       = errors.New("Storage label already in use by another account")
	ErrInvalidLicenseKey       = errors.New("Invalid license key")
	ErrProfileImageMissing     = errors.New("Can't delete a missing profile Image")
	ErrProfileImageUnprocessed = errors.New("Unable to process profile image")
	ErrValidation              = errors.New("Validation failed")
	ErrBodyInvalid             = errors.New("Invalid request body")
	ErrQuotaExceeded           = errors.New("Quota has been exceeded!")
	ErrLivePhotoNotFound       = errors.New("Live photo video not found")
	ErrLivePhotoNotVideo       = errors.New("Live photo video must be a video")
	ErrLivePhotoNotOwned       = errors.New("Live photo video does not belong to the user")
	ErrFileRequired            = errors.New("File is required")
	ErrAssetDataEmpty          = errors.New("Field(s) assetData should not be empty")
	ErrAssetNotFound           = errors.New("Asset not found")
	ErrFieldTooLong            = errors.New("Field value too long")
	ErrUnexpectedEndOfForm     = errors.New("Multipart: Unexpected end of form")
	ErrUnlinkAndroidMotion     = errors.New("Cannot unlink Android motion photos")
	ErrElevatedRequired        = errors.New("Elevated permission is required")
	ErrBothAssetsMustExist     = errors.New("Both assets must exist")
	ErrSameSourceAndTarget     = errors.New("Source and target id must be distinct")
	ErrDuplicateItems          = errors.New("Duplicate items are not allowed")
	ErrMetadataNotFound        = errors.New("Metadata with key")
	ErrSyncAPIKey              = errors.New("Sync endpoints cannot be used with API keys")
	ErrSyncTypeDeprecated      = errors.New("SyncRequestType.")
	ErrInvalidAckType          = errors.New("Invalid ack type:")
	ErrRealtimeTranscoding     = errors.New("Real-time transcoding is not enabled")
	ErrSmartSearchDisabled     = errors.New("Smart search is not enabled")
	ErrInvalidCursor           = errors.New("Invalid cursor")
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
