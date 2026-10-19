package request

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tty.fail/mrus/korda/http/api/problem"
	adminusers "tty.fail/mrus/korda/logic/root/admin/users"
	"tty.fail/mrus/korda/logic/root/apikeys"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	"tty.fail/mrus/korda/logic/root/sessions"
	"tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/models/permission"
	"tty.fail/mrus/korda/models/preferences"
)

const (
	emailSource = `/^[\\p{L}\\p{M}\\p{N}.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[\\p{L}\\p{N}](?:[\\p{L}\\p{M}\\p{N}-]{0,61}[\\p{L}\\p{M}\\p{N}])?(?:\\.[\\p{L}\\p{N}](?:[\\p{L}\\p{M}\\p{N}-]{0,61}[\\p{L}\\p{M}\\p{N}])?)*$/u`
	uuidSource  = `/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$/`
)

func issuesOf(t *testing.T, err error) string {
	t.Helper()

	var validation *problem.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error = %v, want a validation error", err)
	}

	encoded, encodeErr := problem.Encode(validation.Issues)
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}

	return string(encoded)
}

func bindJSON(t *testing.T, body string, dst any) error {
	t.Helper()

	if body == "" {
		return bind(nil, false, dst, modeBody)
	}

	raw, present, err := decodeJSON([]byte(body))
	if err != nil {
		return err
	}

	return bind(raw, present, dst, modeBody)
}

func TestBodyIssuesMatchImmich(t *testing.T) {
	permissionValues, _ := json.Marshal(permission.Values())
	permissionOptions := quoted(permission.Values())
	permissionMessage, _ := json.Marshal("Invalid option: expected one of " + permissionOptions)

	tests := []struct {
		name string
		body string
		dst  func() any
		want string
	}{
		{
			name: "missing password",
			body: `{"email":"admin@example.com"}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["password"],"message":"Invalid input: expected string, received undefined"}]`,
		},
		{
			name: "invalid email",
			body: `{"email":"not-an-email","password":"x"}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"origin":"string","code":"invalid_format","format":"email","pattern":"` + emailSource + `","path":["email"],"message":"Invalid input: expected email, received string"}]`,
		},
		{
			name: "numeric password",
			body: `{"email":"ADMIN@Example.com","password":42}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["password"],"message":"Invalid input: expected string, received number"}]`,
		},
		{
			name: "empty login",
			body: `{}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["email"],"message":"Invalid input: expected email, received undefined"},{"expected":"string","code":"invalid_type","path":["password"],"message":"Invalid input: expected string, received undefined"}]`,
		},
		{
			name: "numeric email",
			body: `{"email":5,"password":"x"}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["email"],"message":"Invalid input: expected email, received number"}]`,
		},
		{
			name: "null email",
			body: `{"email":null,"password":"x"}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["email"],"message":"Invalid input: expected email, received object"}]`,
		},
		{
			name: "array email",
			body: `{"email":["a"],"password":"x"}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["email"],"message":"Invalid input: expected email, received object"}]`,
		},
		{
			name: "null password",
			body: `{"email":"a@b.c","password":null}`,
			dst:  func() any { return new(logicauth.LoginInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["password"],"message":"Invalid input: expected string, received null"}]`,
		},
		{
			name: "string duration",
			body: `{"duration":"x"}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"number","code":"invalid_type","path":["duration"],"message":"Invalid input: expected number, received string"}]`,
		},
		{
			name: "fractional duration",
			body: `{"duration":1.5}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"int","format":"safeint","code":"invalid_type","path":["duration"],"message":"Invalid input: expected int, received number"}]`,
		},
		{
			name: "null duration",
			body: `{"duration":null}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"number","code":"invalid_type","path":["duration"],"message":"Invalid input: expected number, received null"}]`,
		},
		{
			name: "zero duration",
			body: `{"duration":0}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["duration"],"message":"Too small: expected number to be >=1"}]`,
		},
		{
			name: "unsafe duration",
			body: `{"duration":9007199254740993}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"code":"too_big","maximum":9007199254740991,"note":"Integers must be within the safe integer range.","origin":"int","inclusive":true,"path":["duration"],"message":"Too big: expected int to be <=9007199254740991"}]`,
		},
		{
			name: "unsafe negative duration",
			body: `{"duration":-9007199254740993}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"code":"too_small","minimum":-9007199254740991,"note":"Integers must be within the safe integer range.","origin":"int","inclusive":true,"path":["duration"],"message":"Too small: expected int to be >=-9007199254740991"},{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["duration"],"message":"Too small: expected number to be >=1"}]`,
		},
		{
			name: "infinite duration",
			body: `{"duration":1e400}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"number","code":"invalid_type","received":"Infinity","path":["duration"],"message":"Invalid input: expected number, received number"}]`,
		},
		{
			name: "numeric device",
			body: `{"deviceOS":5}`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["deviceOS"],"message":"Invalid input: expected string, received number"}]`,
		},
		{
			name: "array body",
			body: `[]`,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received array"}]`,
		},
		{
			name: "missing body",
			body: ``,
			dst:  func() any { return new(sessions.CreateInput) },
			want: `[{"expected":"object","code":"invalid_type","path":[],"message":"Invalid input: expected object, received undefined"}]`,
		},
		{
			name: "nested preferences",
			body: `{"memories":{"duration":0,"enabled":"yes"},"albums":{"defaultAssetOrder":"up"},"extra":1}`,
			dst:  func() any { return new(preferences.Update) },
			want: `[{"code":"invalid_value","values":["asc","desc"],"path":["albums","defaultAssetOrder"],"message":"Invalid option: expected one of \"asc\"|\"desc\""},{"expected":"boolean","code":"invalid_type","path":["memories","enabled"],"message":"Invalid input: expected boolean, received string"},{"origin":"number","code":"too_small","minimum":1,"inclusive":true,"path":["memories","duration"],"message":"Too small: expected number to be >=1"}]`,
		},
		{
			name: "string preferences section",
			body: `{"memories":"x"}`,
			dst:  func() any { return new(preferences.Update) },
			want: `[{"expected":"object","code":"invalid_type","path":["memories"],"message":"Invalid input: expected object, received string"}]`,
		},
		{
			name: "null preferences section",
			body: `{"albums":null}`,
			dst:  func() any { return new(preferences.Update) },
			want: `[{"expected":"object","code":"invalid_type","path":["albums"],"message":"Invalid input: expected object, received null"}]`,
		},
		{
			name: "numeric avatar color",
			body: `{"avatar":{"color":5}}`,
			dst:  func() any { return new(preferences.Update) },
			want: `[{"code":"invalid_value","values":["primary","pink","red","yellow","blue","green","purple","orange","gray","amber"],"path":["avatar","color"],"message":"Invalid option: expected one of \"primary\"|\"pink\"|\"red\"|\"yellow\"|\"blue\"|\"green\"|\"purple\"|\"orange\"|\"gray\"|\"amber\""}]`,
		},
		{
			name: "null avatar color",
			body: `{"avatar":{"color":null}}`,
			dst:  func() any { return new(preferences.Update) },
			want: `[{"code":"invalid_value","values":["primary","pink","red","yellow","blue","green","purple","orange","gray","amber"],"path":["avatar","color"],"message":"Invalid option: expected one of \"primary\"|\"pink\"|\"red\"|\"yellow\"|\"blue\"|\"green\"|\"purple\"|\"orange\"|\"gray\"|\"amber\""}]`,
		},
		{
			name: "admin create",
			body: `{"email":"a@b.c","password":"p","name":"n","quotaSizeInBytes":-1,"pinCode":"12"}`,
			dst:  func() any { return new(adminusers.CreateInput) },
			want: `[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^\\d{6}$/","path":["pinCode"],"message":"Invalid string: must match pattern /^\\d{6}$/"},{"origin":"number","code":"too_small","minimum":0,"inclusive":true,"path":["quotaSizeInBytes"],"message":"Too small: expected number to be >=0"}]`,
		},
		{
			name: "string permissions",
			body: `{"permissions":"all"}`,
			dst:  func() any { return new(apikeys.CreateInput) },
			want: `[{"expected":"array","code":"invalid_type","path":["permissions"],"message":"Invalid input: expected array, received string"}]`,
		},
		{
			name: "empty permissions",
			body: `{"permissions":[]}`,
			dst:  func() any { return new(apikeys.CreateInput) },
			want: `[{"origin":"array","code":"too_small","minimum":1,"inclusive":true,"path":["permissions"],"message":"Too small: expected array to have >=1 items"}]`,
		},
		{
			name: "missing permissions",
			body: `{"name":"none"}`,
			dst:  func() any { return new(apikeys.CreateInput) },
			want: `[{"expected":"array","code":"invalid_type","path":["permissions"],"message":"Invalid input: expected array, received undefined"}]`,
		},
		{
			name: "unknown permissions",
			body: `{"permissions":[1,"all",null]}`,
			dst:  func() any { return new(apikeys.CreateInput) },
			want: `[{"code":"invalid_value","values":` + string(permissionValues) + `,"path":["permissions",0],"message":` + string(permissionMessage) + `},{"code":"invalid_value","values":` + string(permissionValues) + `,"path":["permissions",2],"message":` + string(permissionMessage) + `}]`,
		},
		{
			name: "short password",
			body: `{"password":"vera-password","newPassword":"short"}`,
			dst:  func() any { return new(logicauth.ChangePasswordInput) },
			want: `[{"origin":"string","code":"too_small","minimum":8,"inclusive":true,"path":["newPassword"],"message":"Too small: expected string to have >=8 characters"}]`,
		},
		{
			name: "short accented password",
			body: `{"password":"x","newPassword":"ééééééé"}`,
			dst:  func() any { return new(logicauth.ChangePasswordInput) },
			want: `[{"origin":"string","code":"too_small","minimum":8,"inclusive":true,"path":["newPassword"],"message":"Too small: expected string to have >=8 characters"}]`,
		},
		{
			name: "short pin",
			body: `{"pinCode":"12345"}`,
			dst:  func() any { return new(logicauth.PinCodeSetupInput) },
			want: `[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^\\d{6}$/","path":["pinCode"],"message":"Invalid string: must match pattern /^\\d{6}$/"}]`,
		},
		{
			name: "numeric pin",
			body: `{"pinCode":123456}`,
			dst:  func() any { return new(logicauth.PinCodeSetupInput) },
			want: `[{"expected":"string","code":"invalid_type","path":["pinCode"],"message":"Invalid input: expected string, received number"}]`,
		},
		{
			name: "license",
			body: `{"licenseKey":"nope","activationKey":"x"}`,
			dst:  func() any { return new(users.LicenseInput) },
			want: `[{"origin":"string","code":"invalid_format","format":"regex","pattern":"/^IM(SV|CL)(-[\\dA-Za-z]{4}){8}$/","path":["licenseKey"],"message":"Invalid string: must match pattern /^IM(SV|CL)(-[\\dA-Za-z]{4}){8}$/"}]`,
		},
		{
			name: "unknown avatar color",
			body: `{"avatarColor":"magenta"}`,
			dst:  func() any { return new(users.UpdateMeInput) },
			want: `[{"code":"invalid_value","values":["primary","pink","red","yellow","blue","green","purple","orange","gray","amber"],"path":["avatarColor"],"message":"Invalid option: expected one of \"primary\"|\"pink\"|\"red\"|\"yellow\"|\"blue\"|\"green\"|\"purple\"|\"orange\"|\"gray\"|\"amber\""}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := issuesOf(t, bindJSON(t, tt.body, tt.dst())); got != tt.want {
				t.Errorf("issues\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}

func TestBodyNormalizesAcceptedValues(t *testing.T) {
	var login logicauth.LoginInput
	if err := bindJSON(t, `{"email":"Admin@Example.com","password":"pw","extra":true}`, &login); err != nil {
		t.Fatal(err)
	}
	if login.Email != "admin@example.com" || login.Password != "pw" {
		t.Errorf("login = %+v", login)
	}

	var session sessions.CreateInput
	if err := bindJSON(t, `{"duration":2.0}`, &session); err != nil {
		t.Fatal(err)
	}
	if session.Duration == nil || *session.Duration != 2 {
		t.Errorf("duration = %v, want 2", session.Duration)
	}

	var created adminusers.CreateInput
	if err := bindJSON(t, `{"email":"juno@example.com","password":"p","name":"Juno","storageLabel":"ju/no","quotaSizeInBytes":null,"avatarColor":null}`, &created); err != nil {
		t.Fatal(err)
	}
	if created.StorageLabel.V != "juno" || !created.StorageLabel.Valid() {
		t.Errorf("storageLabel = %+v, want juno", created.StorageLabel)
	}
	if !created.QuotaSizeInBytes.Set || !created.QuotaSizeInBytes.Null {
		t.Errorf("quotaSizeInBytes = %+v, want an explicit null", created.QuotaSizeInBytes)
	}
	if !created.AvatarColor.Set || !created.AvatarColor.Null {
		t.Errorf("avatarColor = %+v, want an explicit null", created.AvatarColor)
	}
	if created.PinCode.Set {
		t.Errorf("pinCode = %+v, want unset", created.PinCode)
	}

	var change logicauth.ChangePasswordInput
	if err := bindJSON(t, `{"password":"x","newPassword":"😀😀😀😀"}`, &change); err != nil {
		t.Errorf("four emoji are eight UTF-16 units: %v", err)
	}

	var update apikeys.UpdateInput
	if err := bindJSON(t, `{"name":"renamed"}`, &update); err != nil || update.Permissions != nil {
		t.Errorf("update = %+v, %v; want permissions left out", update, err)
	}
}

func TestQueryIssuesMatchImmich(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{
			name: "stringbool",
			raw:  map[string]any{"withDeleted": "yes"},
			want: `[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["withDeleted"],"message":"Invalid option: expected one of \"true\"|\"false\""}]`,
		},
		{
			name: "empty stringbool",
			raw:  map[string]any{"withDeleted": ""},
			want: `[{"code":"invalid_value","expected":"stringbool","values":["true","false"],"path":["withDeleted"],"message":"Invalid option: expected one of \"true\"|\"false\""}]`,
		},
		{
			name: "uuid",
			raw:  map[string]any{"id": "nope"},
			want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"` + uuidSource + `","path":["id"],"message":"Invalid UUID"}]`,
		},
		{
			name: "uuid v7",
			raw:  map[string]any{"id": "00000000-0000-7000-8000-000000000000"},
			want: `[{"origin":"string","code":"invalid_format","format":"uuid","pattern":"` + uuidSource + `","path":["id"],"message":"Invalid UUID"}]`,
		},
		{
			name: "repeated",
			raw:  map[string]any{"withDeleted": []any{"true", "false"}},
			want: `[{"expected":"string","code":"invalid_type","path":["withDeleted"],"message":"Invalid input: expected string, received array"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dst adminusers.SearchInput
			if got := issuesOf(t, bind(tt.raw, true, &dst, modeQuery)); got != tt.want {
				t.Errorf("issues\n got %s\nwant %s", got, tt.want)
			}
		})
	}

	var dst adminusers.SearchInput
	if err := bind(map[string]any{"withDeleted": "true", "id": "757a6107-415e-433f-8a6d-57c33d200dbc"}, true, &dst, modeQuery); err != nil {
		t.Fatal(err)
	}
	if dst.WithDeleted == nil || !*dst.WithDeleted || dst.ID == nil {
		t.Errorf("search = %+v", dst)
	}
}

func TestMalformedBodies(t *testing.T) {
	tests := map[string]string{
		`"x"`:      `Unexpected token '"', ""x"" is not valid JSON`,
		`not json`: `Unexpected token 'n', "not json" is not valid JSON`,
	}

	for body, want := range tests {
		_, _, err := decodeJSON([]byte(body))
		var explicit *problem.Error
		if !errors.As(err, &explicit) || explicit.Status != 400 || explicit.Message != want {
			t.Errorf("decodeJSON(%s) = %v, want 400 %q", body, err, want)
		}
	}

	if _, _, err := decodeJSON([]byte(`{"a":}`)); err == nil || !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("decodeJSON of a broken object = %v", err)
	}
	if _, _, err := decodeJSON([]byte(`{} x`)); err == nil {
		t.Error("decodeJSON accepted trailing data")
	}
}
