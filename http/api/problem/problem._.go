package problem

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	ContentType    = "application/json; charset=utf-8"
	OperationLocal = "korda_operation"
)

type Body struct {
	Message string `json:"message"`
	Errors  []any  `json:"errors,omitempty"`
}

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

type ValidationError struct {
	Issues []any
}

func (e *ValidationError) Error() string {
	return errs.ErrValidation.Error()
}

func (e *ValidationError) Unwrap() error {
	return errs.ErrValidation
}

type mapping struct {
	err    error
	status int
}

var mappings = []mapping{
	{errs.ErrAuthenticationRequired, fiber.StatusUnauthorized},
	{errs.ErrInvalidUserToken, fiber.StatusUnauthorized},
	{errs.ErrInvalidAPIKey, fiber.StatusUnauthorized},
	{errs.ErrInvalidShareKey, fiber.StatusUnauthorized},
	{errs.ErrInvalidShareSlug, fiber.StatusUnauthorized},
	{errs.ErrIncorrectCredentials, fiber.StatusUnauthorized},
	{errs.ErrUnauthorized, fiber.StatusUnauthorized},
	{errs.ErrElevatedRequired, fiber.StatusUnauthorized},
	{errs.ErrForbidden, fiber.StatusForbidden},
	{errs.ErrMissingPermission, fiber.StatusForbidden},
	{errs.ErrNotAPIKey, fiber.StatusForbidden},
	{errs.ErrCannotDeleteOwnAccount, fiber.StatusForbidden},
	{errs.ErrAdminSetupNotAvailable, fiber.StatusBadRequest},
	{errs.ErrEmailNotAvailable, fiber.StatusBadRequest},
	{errs.ErrPasswordRequired, fiber.StatusBadRequest},
	{errs.ErrFirstUserMustBeAdmin, fiber.StatusBadRequest},
	{errs.ErrUserNotFound, fiber.StatusBadRequest},
	{errs.ErrWrongPassword, fiber.StatusBadRequest},
	{errs.ErrWrongPinCode, fiber.StatusBadRequest},
	{errs.ErrPinCodeExists, fiber.StatusBadRequest},
	{errs.ErrPinCodeMissing, fiber.StatusBadRequest},
	{errs.ErrPasswordOrPinRequired, fiber.StatusBadRequest},
	{errs.ErrSessionTokenRequired, fiber.StatusBadRequest},
	{errs.ErrNoFieldsToUpdate, fiber.StatusBadRequest},
	{errs.ErrNoAccess, fiber.StatusBadRequest},
	{errs.ErrAPIKeyNotFound, fiber.StatusBadRequest},
	{errs.ErrCannotGrantPermissions, fiber.StatusBadRequest},
	{errs.ErrCannotRotateAPIKey, fiber.StatusBadRequest},
	{errs.ErrAdminStatusSelf, fiber.StatusBadRequest},
	{errs.ErrStorageLabelInUse, fiber.StatusBadRequest},
	{errs.ErrInvalidLicenseKey, fiber.StatusBadRequest},
	{errs.ErrProfileImageMissing, fiber.StatusBadRequest},
	{errs.ErrProfileImageUnprocessed, fiber.StatusBadRequest},
	{errs.ErrUnsupportedFileType, fiber.StatusBadRequest},
	{errs.ErrFileEmpty, fiber.StatusBadRequest},
	{errs.ErrUnexpectedFileField, fiber.StatusBadRequest},
	{errs.ErrQuotaExceeded, fiber.StatusBadRequest},
	{errs.ErrLivePhotoNotFound, fiber.StatusBadRequest},
	{errs.ErrLivePhotoNotVideo, fiber.StatusBadRequest},
	{errs.ErrLivePhotoNotOwned, fiber.StatusBadRequest},
	{errs.ErrFileRequired, fiber.StatusBadRequest},
	{errs.ErrAssetDataEmpty, fiber.StatusBadRequest},
	{errs.ErrAssetNotFound, fiber.StatusBadRequest},
	{errs.ErrFieldTooLong, fiber.StatusBadRequest},
	{errs.ErrUnexpectedEndOfForm, fiber.StatusBadRequest},
	{errs.ErrUnlinkAndroidMotion, fiber.StatusBadRequest},
	{errs.ErrBothAssetsMustExist, fiber.StatusBadRequest},
	{errs.ErrSameSourceAndTarget, fiber.StatusBadRequest},
	{errs.ErrDuplicateItems, fiber.StatusBadRequest},
	{errs.ErrMetadataNotFound, fiber.StatusBadRequest},
	{errs.ErrRealtimeTranscoding, fiber.StatusBadRequest},
	{errs.ErrSmartSearchDisabled, fiber.StatusBadRequest},
	{errs.ErrInvalidCursor, fiber.StatusBadRequest},
	{errs.ErrNotFound, fiber.StatusNotFound},
	{errs.ErrAssetMediaNotFound, fiber.StatusNotFound},
	{errs.ErrStreamNotReady, fiber.StatusNotFound},
	{errs.ErrPayloadTooLarge, fiber.StatusRequestEntityTooLarge},
	{errs.ErrTooManyRequests, fiber.StatusTooManyRequests},
}

var upper = regexp.MustCompile(`[A-Z]+`)

func Failure(operation string) string {
	if operation == "" {
		return errs.ErrInternal.Error()
	}

	return "Failed to " + upper.ReplaceAllStringFunc(operation, func(s string) string {
		return " " + strings.ToLower(s)
	})
}

func From(err error, operation string) (status int, body Body, known bool) {
	if explicit, ok := errors.AsType[*Error](err); ok {
		return explicit.Status, Body{Message: explicit.Message}, true
	}

	if validation, ok := errors.AsType[*ValidationError](err); ok {
		return fiber.StatusBadRequest, Body{Message: validation.Error(), Errors: validation.Issues}, true
	}

	for _, m := range mappings {
		if errors.Is(err, m.err) {
			return m.status, Body{Message: err.Error()}, true
		}
	}

	return fiber.StatusInternalServerError, Body{Message: Failure(operation)}, false
}

func fromFiber(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if !errors.As(err, &fe) {
		return err
	}

	switch fe.Code {
	case fiber.StatusNotFound, fiber.StatusMethodNotAllowed:
		return &Error{Status: fiber.StatusNotFound, Message: "Cannot " + c.Method() + " " + c.OriginalURL()}
	case fiber.StatusRequestEntityTooLarge:
		return errs.ErrPayloadTooLarge
	}

	return &Error{Status: fe.Code, Message: fe.Message}
}

func Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func Respond(c fiber.Ctx, rt *runtime.Runtime, err error) error {
	err = fromFiber(c, err)

	operation, _ := c.Locals(OperationLocal).(string)
	status, body, known := From(err, operation)
	if !known && rt != nil {
		rt.Error("api", "error", "path", c.Path(), "operation", operation, "error", err)
	}

	encoded, encodeErr := Encode(body)
	if encodeErr != nil {
		return encodeErr
	}

	c.Status(status)
	c.Set(fiber.HeaderContentType, ContentType)
	return c.Send(encoded)
}
