package problem

import (
	"errors"
	"fmt"
	"testing"

	"tty.fail/mrus/korda/errs"
)

func TestFromKeepsImmichsMessages(t *testing.T) {
	tests := []struct {
		err     error
		status  int
		message string
	}{
		{errs.ErrAuthenticationRequired, 401, "Authentication required"},
		{fmt.Errorf("%w: %s", errs.ErrMissingPermission, "session.read"), 403, "Missing required permission: session.read"},
		{fmt.Errorf("%w %s access", errs.ErrNoAccess, "authDevice.delete"), 400, "Not found or no authDevice.delete access"},
		{fmt.Errorf("%w not found", errs.ErrAPIKeyNotFound), 400, "API Key not found not found"},
		{fmt.Errorf("%w %s", errs.ErrUnsupportedFileType, "a.gif"), 400, "Unsupported file type a.gif"},
		{errs.ErrNotFound, 404, "Not Found"},
		{errs.ErrPayloadTooLarge, 413, "Payload Too Large"},
		{&Error{Status: 404, Message: "Cannot GET /api/nope"}, 404, "Cannot GET /api/nope"},
		{errors.New("disk on fire"), 500, "Internal server error"},
	}

	for _, tt := range tests {
		status, body, _ := From(tt.err, "")
		if status != tt.status || body.Message != tt.message {
			t.Errorf("From(%v) = %d %q, want %d %q", tt.err, status, body.Message, tt.status, tt.message)
		}
	}
}

func TestFromNamesTheFailedOperation(t *testing.T) {
	tests := map[string]string{
		"createProfileImage": "Failed to create profile image",
		"getMyUser":          "Failed to get my user",
		"getApiKeys":         "Failed to get api keys",
		"pingServer":         "Failed to ping server",
		"":                   "Internal server error",
	}

	for operation, want := range tests {
		status, body, known := From(errors.New("boom"), operation)
		if status != 500 || body.Message != want || known {
			t.Errorf("From(boom, %q) = %d %q %v, want 500 %q false", operation, status, body.Message, known, want)
		}
	}
}

func TestFromRendersValidationIssues(t *testing.T) {
	status, body, known := From(fmt.Errorf("bind: %w", &ValidationError{Issues: []any{"issue"}}), "")
	if status != 400 || body.Message != "Validation failed" || len(body.Errors) != 1 || !known {
		t.Errorf("From(validation) = %d %+v %v", status, body, known)
	}
}

func TestEncodeKeepsHTMLAndDropsTheNewline(t *testing.T) {
	encoded, err := Encode(Body{Message: "Cannot GET /api/nope?x=<b>&y"})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"message":"Cannot GET /api/nope?x=<b>&y"}` {
		t.Errorf("Encode = %s", encoded)
	}
}
