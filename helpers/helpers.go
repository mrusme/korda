package helpers

import (
	"context"

	"tty.fail/mrus/korda/services/events"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/media"
	"tty.fail/mrus/korda/services/mediaroot"
	"tty.fail/mrus/korda/services/repositories"
	"tty.fail/mrus/korda/services/websocket"
	"xn--gckvb8fzb.com/glides/runtime"
)

func Repositories(rt *runtime.Runtime) *repositories.Repositories {
	if srv, _ := rt.GetService("repositories"); srv != nil {
		return srv.(*repositories.Repositories)
	}
	return nil
}

func MediaRoot(rt *runtime.Runtime) *mediaroot.MediaRoot {
	if srv, _ := rt.GetService("mediaroot"); srv != nil {
		return srv.(*mediaroot.MediaRoot)
	}
	return nil
}

func Media(rt *runtime.Runtime) *media.Media {
	if srv, _ := rt.GetService("media"); srv != nil {
		return srv.(*media.Media)
	}
	return nil
}

func Jobs(rt *runtime.Runtime) *jobs.Jobs {
	if srv, _ := rt.GetService("jobs"); srv != nil {
		return srv.(*jobs.Jobs)
	}
	return nil
}

func Websocket(rt *runtime.Runtime) *websocket.Hub {
	if srv, _ := rt.GetService("websocket"); srv != nil {
		return srv.(*websocket.Hub)
	}
	return nil
}

func Events(rt *runtime.Runtime) *events.Bus {
	if srv, _ := rt.GetService("events"); srv != nil {
		return srv.(*events.Bus)
	}
	return nil
}

func Emit(ctx context.Context, rt *runtime.Runtime, event any) {
	if bus := Events(rt); bus != nil {
		bus.Emit(ctx, event)
	}
}
