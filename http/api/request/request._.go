package request

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/problem"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/models/auth"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	ContentType    = problem.ContentType
	OperationLocal = problem.OperationLocal
	JSONLimit      = 10 << 20

	CookieAccessToken     = "immich_access_token"
	CookieAuthType        = "immich_auth_type"
	CookieIsAuthenticated = "immich_is_authenticated"
	CookieSharedLinkToken = "immich_shared_link_token"
)

type Request struct {
	c         fiber.Ctx
	rt        *runtime.Runtime
	operation string
	Auth      *auth.Auth
}

func New(rt *runtime.Runtime, c fiber.Ctx, operation string) *Request {
	c.Locals(OperationLocal, operation)

	return &Request{c: c, rt: rt, operation: operation}
}

func (req *Request) Ctx() fiber.Ctx {
	return req.c
}

func (req *Request) Context() context.Context {
	return req.c.Context()
}

func (req *Request) Runtime() *runtime.Runtime {
	return req.rt
}

func first(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

func bearer(header string) string {
	parts := strings.SplitN(header, " ", 3)
	if len(parts) >= 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}

	return ""
}

func (req *Request) Credentials() logicauth.Credentials {
	c := req.c

	return logicauth.Credentials{
		ShareKey:  first(c.Get("x-immich-share-key"), c.Query("key")),
		ShareSlug: first(c.Get("x-immich-share-slug"), c.Query("slug")),
		SessionToken: first(
			c.Get("x-immich-user-token"),
			c.Get("x-immich-session-token"),
			c.Query("sessionKey"),
			bearer(c.Get(fiber.HeaderAuthorization)),
			c.Cookies(CookieAccessToken),
		),
		APIKey:      first(c.Get("x-api-key"), c.Query("apiKey")),
		UserAgent:   c.Get(fiber.HeaderUserAgent),
		DeviceModel: c.Get("devicemodel"),
		DeviceType:  c.Get("devicetype"),
	}
}

func (req *Request) LoginDetails() logicauth.LoginDetails {
	return logicauth.LoginDetails{
		ClientIP:    req.c.IP(),
		UserAgent:   req.c.Get(fiber.HeaderUserAgent),
		DeviceModel: req.c.Get("devicemodel"),
		DeviceType:  req.c.Get("devicetype"),
	}
}

func (req *Request) guard(check func() error) error {
	req.c.Locals(OperationLocal, "")
	if err := check(); err != nil {
		return err
	}
	req.c.Locals(OperationLocal, req.operation)

	return nil
}

func (req *Request) Require(r logicauth.Requirement) error {
	return req.guard(func() (err error) {
		req.Auth, err = logicauth.Authenticate(req.Context(), req.rt, req.Credentials(), r, req.c.Path())
		return err
	})
}

func (req *Request) RequireSetup() error {
	return req.guard(func() error {
		return logicauth.RequireSetupAvailable(req.Context(), req.rt)
	})
}

func (req *Request) UUID(name string) (uuid.UUID, error) {
	value := req.c.Params(name)

	w := &walker{mode: modeQuery}
	w.string([]any{name}, value, true, rules{uuid: true})
	if len(w.issues) > 0 {
		return uuid.Nil, Validation(w.issues...)
	}

	return uuid.Parse(value)
}

func (req *Request) Params(dst any, names ...string) error {
	raw := NewObject()
	for _, name := range names {
		raw.Set(name, req.c.Params(name))
	}

	return bind(raw, true, dst, modeQuery)
}

func (req *Request) Headers(dst any, names ...string) error {
	raw := NewObject()
	for _, name := range names {
		if value := req.c.Get(name); value != "" {
			raw.Set(name, value)
		}
	}

	return bind(raw, true, dst, modeQuery)
}

func (req *Request) Query(dst any) error {
	raw := NewObject()
	for key, value := range req.c.RequestCtx().QueryArgs().All() {
		appendField(raw, string(key), string(value))
	}

	return bind(raw, true, dst, modeQuery)
}

func AppendField(o *Object, name string, value string) {
	appendField(o, name, value)
}

func MissingBody() Issue {
	iss, message := invalidType("object", "undefined")
	return iss.withPath([]any{}, message)
}

func appendField(o *Object, name string, value string) {
	existing, ok := o.Get(name)
	switch current := existing.(type) {
	case string:
		o.Set(name, []any{current, value})
	case []any:
		o.Set(name, append(current, value))
	default:
		if !ok {
			o.Set(name, value)
		}
	}
}

func (req *Request) Body(dst any) error {
	raw, present, err := req.body()
	if err != nil {
		return err
	}

	return bind(raw, present, dst, modeBody)
}

func (req *Request) body() (any, bool, error) {
	mediaType, _, _ := mime.ParseMediaType(req.c.Get(fiber.HeaderContentType))

	switch mediaType {
	case fiber.MIMEApplicationJSON:
		b, err := req.Read(JSONLimit)
		if err != nil || len(b) == 0 {
			return nil, false, err
		}
		return decodeJSON(b)

	case fiber.MIMEApplicationForm:
		b, err := req.Read(JSONLimit)
		if err != nil || len(b) == 0 {
			return nil, false, err
		}
		return decodeForm(b), true, nil
	}

	return nil, false, nil
}

func (req *Request) Read(limit int64) ([]byte, error) {
	r := req.c.Request()
	if n := r.Header.ContentLength(); int64(n) > limit {
		return nil, errs.ErrPayloadTooLarge
	}

	stream := r.BodyStream()
	if stream == nil {
		b := r.Body()
		if int64(len(b)) > limit {
			return nil, errs.ErrPayloadTooLarge
		}
		return b, nil
	}

	b, err := io.ReadAll(io.LimitReader(stream, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, errs.ErrPayloadTooLarge
	}

	return b, nil
}

func (req *Request) Stream() io.Reader {
	r := req.c.Request()
	if stream := r.BodyStream(); stream != nil {
		return stream
	}

	return bytes.NewReader(r.Body())
}

func decodeJSON(b []byte) (any, bool, error) {
	trimmed := bytes.TrimLeft(b, " \t\r\n")
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		token := ""
		if len(trimmed) > 0 {
			token = string(trimmed[:1])
		}
		return nil, false, &problem.Error{
			Status:  fiber.StatusBadRequest,
			Message: "Unexpected token '" + token + "', \"" + string(b) + "\" is not valid JSON",
		}
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	var check any
	if err := dec.Decode(&check); err != nil {
		return nil, false, &problem.Error{Status: fiber.StatusBadRequest, Message: err.Error()}
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, false, &problem.Error{Status: fiber.StatusBadRequest, Message: "Unexpected non-whitespace character after JSON"}
	}

	raw, err := decodeOrdered(b)
	if err != nil {
		return nil, false, &problem.Error{Status: fiber.StatusBadRequest, Message: err.Error()}
	}

	return raw, true, nil
}

func decodeForm(b []byte) *Object {
	var args fasthttp.Args
	args.ParseBytes(b)

	raw := NewObject()
	for key, value := range args.All() {
		appendField(raw, string(key), string(value))
	}

	return raw
}

func Bind(raw any, dst any) error {
	return bind(raw, true, dst, modeQuery)
}

func bind(raw any, present bool, dst any, m mode) error {
	t := reflect.TypeOf(dst)
	if t == nil || t.Kind() != reflect.Pointer {
		return fmt.Errorf("%w: %T", ErrUnsupportedType, dst)
	}

	v, issues, err := validate(raw, present, t.Elem(), m)
	if err != nil {
		return err
	}
	if len(issues) > 0 {
		out := make([]any, len(issues))
		for i, iss := range issues {
			out[i] = iss
		}
		return &problem.ValidationError{Issues: out}
	}

	encoded, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return json.Unmarshal(encoded, dst)
}

func Validation(issues ...Issue) error {
	out := make([]any, len(issues))
	for i, iss := range issues {
		out[i] = iss
	}

	return &problem.ValidationError{Issues: out}
}
