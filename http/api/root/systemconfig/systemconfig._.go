package systemconfig

import (
	"bytes"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicsystemconfig "tty.fail/mrus/korda/logic/root/systemconfig"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

func mount(rt *runtime.Runtime, router fiber.Router, name string, register func(r *Route, base fiber.Router)) *Route {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For(name).Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) { register(r, base) }, r.Path+".")

	return r
}

func NewAdmin(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APIAdminConfig", func(r *Route, base fiber.Router) {
		base.Get("", r.GetAdminConfig).Name("getAdminConfig")
		base.Get("/defaults", r.GetAdminConfigDefaults).Name("getAdminConfigDefaults")
		base.Put("", r.UpdateAdminConfig).Name("updateAdminConfig")
	}), nil
}

func NewSystem(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APISystemConfig", func(r *Route, base fiber.Router) {
		base.Get("", r.GetConfig).Name("getConfig")
		base.Get("/defaults", r.GetConfigDefaults).Name("getConfigDefaults")
		base.Put("", r.UpdateConfig).Name("updateConfig")
		base.Get("/storage-template-options", r.GetStorageTemplateOptions).Name("getStorageTemplateOptions")
	}), nil
}

func NewUser(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APIConfig", func(r *Route, base fiber.Router) {
		base.Get("", r.GetUserConfig).Name("getUserConfig")
		base.Get("/defaults", r.GetUserConfigDefaults).Name("getUserConfigDefaults")
	}), nil
}

func NewPublic(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APIPublicConfig", func(r *Route, base fiber.Router) {
		base.Get("", r.GetPublicConfig).Name("getPublicConfig")
		base.Get("/defaults", r.GetPublicConfigDefaults).Name("getPublicConfigDefaults")
	}), nil
}

func NewMetadata(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APISystemMetadata", func(r *Route, base fiber.Router) {
		base.Get("/admin-onboarding", r.GetAdminOnboarding).Name("getAdminOnboarding")
		base.Post("/admin-onboarding", r.UpdateAdminOnboarding).Name("updateAdminOnboarding")
		base.Get("/reverse-geocoding-state", r.GetReverseGeocodingState).Name("getReverseGeocodingState")
		base.Get("/version-check-state", r.GetVersionCheckState).Name("getVersionCheckState")
	}), nil
}

type ReverseGeocodingState struct {
	LastUpdate         *string `json:"lastUpdate"`
	LastImportFileName *string `json:"lastImportFileName"`
}

type VersionCheckState struct {
	CheckedAt      *string `json:"checkedAt"`
	ReleaseVersion *string `json:"releaseVersion"`
}

type Onboarding struct {
	IsOnboarded bool `json:"isOnboarded"`
}

func (r *Route) start(c fiber.Ctx, operation string, need logicauth.Requirement) (*request.Request, error) {
	req := request.New(r.Runtime, c, operation)
	return req, req.Require(need)
}

func admin(p permission.Permission) logicauth.Requirement {
	return logicauth.Requirement{Permission: p, Admin: true}
}

func (r *Route) effective(c fiber.Ctx, operation string, need logicauth.Requirement, body func(*logicsystemconfig.Config) ([]byte, error)) error {
	req, err := r.start(c, operation, need)
	if err != nil {
		return err
	}
	cfg, err := logicsystemconfig.Effective(r.Runtime)
	if err != nil {
		return err
	}
	b, err := body(cfg)
	if err != nil {
		return err
	}

	return req.RespondJSON(fiber.StatusOK, b)
}

func (r *Route) defaults(c fiber.Ctx, operation string, need *logicauth.Requirement, body func(*logicsystemconfig.Config) ([]byte, error)) error {
	req := request.New(r.Runtime, c, operation)
	if need != nil {
		if err := req.Require(*need); err != nil {
			return err
		}
	}
	cfg, err := logicsystemconfig.Defaults()
	if err != nil {
		return err
	}
	b, err := body(cfg)
	if err != nil {
		return err
	}

	return req.RespondJSON(fiber.StatusOK, b)
}

func subset(leaves []string) func(*logicsystemconfig.Config) ([]byte, error) {
	return func(cfg *logicsystemconfig.Config) ([]byte, error) { return view.ConfigSubsetBody(cfg, leaves) }
}

func (r *Route) update(c fiber.Ctx, operation string, p permission.Permission) error {
	req, err := r.start(c, operation, admin(p))
	if err != nil {
		return err
	}

	var in logicsystemconfig.Config
	if err = req.Body(&in); err != nil {
		return err
	}

	return logicsystemconfig.Update(&in)
}

func (r *Route) GetAdminConfig(c fiber.Ctx) error {
	return r.effective(c, "getAdminConfig", admin(permission.AdminConfigRead), view.ConfigBody)
}

func (r *Route) GetConfig(c fiber.Ctx) error {
	return r.effective(c, "getConfig", admin(permission.SystemConfigRead), view.ConfigBody)
}

func (r *Route) GetAdminConfigDefaults(c fiber.Ctx) error {
	need := admin(permission.AdminConfigRead)
	return r.defaults(c, "getAdminConfigDefaults", &need, view.ConfigDefaultsBody)
}

func (r *Route) GetConfigDefaults(c fiber.Ctx) error {
	need := admin(permission.SystemConfigRead)
	return r.defaults(c, "getConfigDefaults", &need, view.ConfigDefaultsBody)
}

func (r *Route) UpdateAdminConfig(c fiber.Ctx) error {
	return r.update(c, "updateAdminConfig", permission.AdminConfigUpdate)
}

func (r *Route) UpdateConfig(c fiber.Ctx) error {
	return r.update(c, "updateConfig", permission.SystemConfigUpdate)
}

func (r *Route) GetStorageTemplateOptions(c fiber.Ctx) error {
	req, err := r.start(c, "getStorageTemplateOptions", admin(permission.SystemConfigRead))
	if err != nil {
		return err
	}

	return req.RespondJSON(fiber.StatusOK, bytes.TrimSpace(logicsystemconfig.StorageTemplateOptions))
}

func (r *Route) GetUserConfig(c fiber.Ctx) error {
	return r.effective(c, "getUserConfig", logicauth.Requirement{Permission: permission.UserConfigRead},
		subset(logicsystemconfig.UserLeaves))
}

func (r *Route) GetUserConfigDefaults(c fiber.Ctx) error {
	need := logicauth.Requirement{Permission: permission.UserConfigRead}
	return r.defaults(c, "getUserConfigDefaults", &need, subset(logicsystemconfig.UserLeaves))
}

func (r *Route) GetPublicConfig(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getPublicConfig")
	cfg, err := logicsystemconfig.Effective(r.Runtime)
	if err != nil {
		return err
	}
	b, err := view.ConfigSubsetBody(cfg, logicsystemconfig.PublicLeaves)
	if err != nil {
		return err
	}

	return req.RespondJSON(fiber.StatusOK, b)
}

func (r *Route) GetPublicConfigDefaults(c fiber.Ctx) error {
	return r.defaults(c, "getPublicConfigDefaults", nil, subset(logicsystemconfig.PublicLeaves))
}

func (r *Route) GetAdminOnboarding(c fiber.Ctx) error {
	req, err := r.start(c, "getAdminOnboarding", admin(permission.SystemMetadataRead))
	if err != nil {
		return err
	}

	onboarded, err := logicsystemconfig.Onboarding(req.Context(), r.Runtime)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, Onboarding{IsOnboarded: onboarded})
}

func (r *Route) UpdateAdminOnboarding(c fiber.Ctx) error {
	req, err := r.start(c, "updateAdminOnboarding", admin(permission.SystemMetadataUpdate))
	if err != nil {
		return err
	}

	var in logicsystemconfig.OnboardingInput
	if err = req.Body(&in); err != nil {
		return err
	}
	if err = logicsystemconfig.SetOnboarding(req.Context(), r.Runtime, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) GetReverseGeocodingState(c fiber.Ctx) error {
	req, err := r.start(c, "getReverseGeocodingState", admin(permission.SystemMetadataRead))
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, ReverseGeocodingState{})
}

func (r *Route) GetVersionCheckState(c fiber.Ctx) error {
	req, err := r.start(c, "getVersionCheckState", admin(permission.SystemMetadataRead))
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, VersionCheckState{})
}
