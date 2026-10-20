package sync

import (
	"bufio"
	"errors"

	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/request"
	"tty.fail/mrus/korda/http/api/view"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicsync "tty.fail/mrus/korda/logic/root/sync"
	"tty.fail/mrus/korda/models/permission"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

const ContentTypeJSONLines = "application/jsonlines+json"

type Route struct {
	route.RouteController
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("APISync").Pathname()
	r.Env = route.NewEnv()

	r.Router.Route("/"+r.Path, func(base fiber.Router) {
		base.Post("/stream", r.GetSyncStream).Name("getSyncStream")
		base.Get("/ack", r.GetSyncAck).Name("getSyncAck")
		base.Post("/ack", r.SendSyncAck).Name("sendSyncAck")
		base.Delete("/ack", r.DeleteSyncAck).Name("deleteSyncAck")
	}, r.Path+".")

	return r, nil
}

func (r *Route) GetSyncStream(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSyncStream")
	if err := req.Require(logicauth.Requirement{Permission: permission.SyncStream}); err != nil {
		return err
	}

	var in logicsync.StreamInput
	if err := req.Body(&in); err != nil {
		return err
	}

	c.Locals(request.OperationLocal, "")
	plan, err := logicsync.Prepare(req.Context(), r.Runtime, req.Auth, &in)
	if err != nil {
		return err
	}

	ctx := c.RequestCtx()
	rt := r.Runtime
	c.Set(fiber.HeaderContentType, ContentTypeJSONLines)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		err := plan.Stream(ctx, rt, func(l logicsync.Line) error {
			line, err := view.EncodeSyncLine(l)
			if err != nil {
				return err
			}
			if _, err = w.Write(line); err != nil {
				return errClientGone
			}
			if err = w.Flush(); err != nil {
				return errClientGone
			}
			return nil
		})
		switch {
		case errors.Is(err, errClientGone):
			rt.Debug("sync", "stream", "client closed the connection")
		case err != nil:
			rt.Error("sync", "stream", "error", err)
			if conn := ctx.Conn(); conn != nil {
				_ = conn.Close()
			}
		}
	})
}

var errClientGone = errors.New("client closed the connection")

func (r *Route) GetSyncAck(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getSyncAck")
	if err := req.Require(logicauth.Requirement{Permission: permission.SyncCheckpointRead}); err != nil {
		return err
	}

	list, err := logicsync.Acks(req.Context(), r.Runtime, req.Auth)
	if err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, view.NewSyncAcks(list))
}

func (r *Route) SendSyncAck(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "sendSyncAck")
	if err := req.Require(logicauth.Requirement{Permission: permission.SyncCheckpointUpdate}); err != nil {
		return err
	}

	var in logicsync.AckSetInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicsync.SetAcks(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}

func (r *Route) DeleteSyncAck(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteSyncAck")
	if err := req.Require(logicauth.Requirement{Permission: permission.SyncCheckpointDelete}); err != nil {
		return err
	}

	var in logicsync.AckDeleteInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicsync.DeleteAcks(req.Context(), r.Runtime, req.Auth, &in); err != nil {
		return err
	}

	return req.NoContent()
}
