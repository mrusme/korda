package workflows

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/logic/helpers/access"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicworkflows "tty.fail/mrus/korda/logic/root/workflows"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

func NewPlugins(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APIPlugins").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Get("", r.SearchPlugins).Name("searchPlugins")
		base.Get("/methods", r.SearchPluginMethods).Name("searchPluginMethods")
		base.Get("/templates", r.SearchPluginTemplates).Name("searchPluginTemplates")
		base.Get("/:id", r.GetPlugin).Name("getPlugin")
	}, r.Path+".")

	return r, nil
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APIWorkflows").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("", r.CreateWorkflow).Name("createWorkflow")
		base.Get("", r.SearchWorkflows).Name("searchWorkflows")
		base.Get("/triggers", r.GetWorkflowTriggers).Name("getWorkflowTriggers")
		base.Get("/:id", r.GetWorkflow).Name("getWorkflow")
		base.Get("/:id/share", r.GetWorkflowForShare).Name("getWorkflowForShare")
		base.Put("/:id", r.UpdateWorkflow).Name("updateWorkflow")
		base.Patch("/:id", r.UpdateWorkflowV3).Name("updateWorkflowV3")
		base.Delete("/:id", r.DeleteWorkflow).Name("deleteWorkflow")
		base.Get("/:id/logs", r.GetWorkflowLogs).Name("getWorkflowLogs")
	}, r.Path+".")

	return r, nil
}

type trigger struct {
	Trigger string   `json:"trigger"`
	Types   []string `json:"types"`
}

var triggers = []trigger{
	{Trigger: "AssetCreate", Types: []string{"AssetV1"}},
	{Trigger: "AssetMetadataExtraction", Types: []string{"AssetV1"}},
	{Trigger: "AssetTagged", Types: []string{"AssetV1"}},
}

func (r *Route) start(c fiber.Ctx, operation string, need logicauth.Requirement) (*request.Request, error) {
	req := request.New(r.Runtime, c, operation)
	return req, req.Require(need)
}

func with(p permission.Permission) logicauth.Requirement {
	return logicauth.Requirement{Permission: p}
}

func (r *Route) SearchPlugins(c fiber.Ctx) error {
	req, err := r.start(c, "searchPlugins", with(permission.PluginRead))
	if err != nil {
		return err
	}

	var in logicworkflows.PluginSearchInput
	if err = req.Query(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) SearchPluginMethods(c fiber.Ctx) error {
	req, err := r.start(c, "searchPluginMethods", with(permission.PluginRead))
	if err != nil {
		return err
	}

	var in logicworkflows.MethodSearchInput
	if err = req.Query(&in); err != nil {
		return err
	}
	if err = logicworkflows.MethodSearch(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) SearchPluginTemplates(c fiber.Ctx) error {
	req, err := r.start(c, "searchPluginTemplates", with(permission.PluginRead))
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) GetPlugin(c fiber.Ctx) error {
	req, err := r.start(c, "getPlugin", with(permission.PluginRead))
	if err != nil {
		return err
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return errs.ErrPluginNotFound
}

func (r *Route) CreateWorkflow(c fiber.Ctx) error {
	req, err := r.start(c, "createWorkflow", with(permission.WorkflowCreate))
	if err != nil {
		return err
	}

	var in logicworkflows.CreateInput
	if err = req.Body(&in); err != nil {
		return err
	}

	return logicworkflows.Create(&in)
}

func (r *Route) SearchWorkflows(c fiber.Ctx) error {
	req, err := r.start(c, "searchWorkflows", with(permission.WorkflowRead))
	if err != nil {
		return err
	}

	var in logicworkflows.SearchInput
	if err = req.Query(&in); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, []struct{}{})
}

func (r *Route) GetWorkflowTriggers(c fiber.Ctx) error {
	req, err := r.start(c, "getWorkflowTriggers", logicauth.Requirement{NoPermission: true})
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, triggers)
}

func (r *Route) missing(c fiber.Ctx, operation string, p permission.Permission, body any, query any) error {
	req, err := r.start(c, operation, with(p))
	if err != nil {
		return err
	}
	if body != nil {
		if err = req.Body(body); err != nil {
			return err
		}
	}
	if query != nil {
		if err = req.Query(query); err != nil {
			return err
		}
	}
	if _, err = req.UUID("id"); err != nil {
		return err
	}

	return access.Denied(p)
}

func (r *Route) GetWorkflow(c fiber.Ctx) error {
	return r.missing(c, "getWorkflow", permission.WorkflowRead, nil, nil)
}

func (r *Route) GetWorkflowForShare(c fiber.Ctx) error {
	return r.missing(c, "getWorkflowForShare", permission.WorkflowRead, nil, nil)
}

func (r *Route) UpdateWorkflow(c fiber.Ctx) error {
	return r.missing(c, "updateWorkflow", permission.WorkflowUpdate, new(logicworkflows.UpdateInput), nil)
}

func (r *Route) UpdateWorkflowV3(c fiber.Ctx) error {
	return r.missing(c, "updateWorkflowV3", permission.WorkflowUpdate, new(logicworkflows.UpdateInput), nil)
}

func (r *Route) DeleteWorkflow(c fiber.Ctx) error {
	return r.missing(c, "deleteWorkflow", permission.WorkflowDelete, nil, nil)
}

func (r *Route) GetWorkflowLogs(c fiber.Ctx) error {
	return r.missing(c, "getWorkflowLogs", permission.WorkflowLogs, nil, new(logicworkflows.LogsInput))
}
