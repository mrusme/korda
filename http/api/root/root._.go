package root

import (
	"github.com/gofiber/fiber/v3"
	"tty.fail/mrus/korda/http/api/root/activities"
	"tty.fail/mrus/korda/http/api/root/admin"
	"tty.fail/mrus/korda/http/api/root/albums"
	"tty.fail/mrus/korda/http/api/root/apikeys"
	"tty.fail/mrus/korda/http/api/root/assetfiles"
	"tty.fail/mrus/korda/http/api/root/assets"
	"tty.fail/mrus/korda/http/api/root/auth"
	"tty.fail/mrus/korda/http/api/root/clustergroups"
	"tty.fail/mrus/korda/http/api/root/download"
	"tty.fail/mrus/korda/http/api/root/duplicates"
	"tty.fail/mrus/korda/http/api/root/faces"
	"tty.fail/mrus/korda/http/api/root/geomap"
	"tty.fail/mrus/korda/http/api/root/jobs"
	"tty.fail/mrus/korda/http/api/root/libraries"
	"tty.fail/mrus/korda/http/api/root/memories"
	"tty.fail/mrus/korda/http/api/root/notifications"
	"tty.fail/mrus/korda/http/api/root/oauth"
	"tty.fail/mrus/korda/http/api/root/partners"
	"tty.fail/mrus/korda/http/api/root/people"
	"tty.fail/mrus/korda/http/api/root/queues"
	"tty.fail/mrus/korda/http/api/root/search"
	"tty.fail/mrus/korda/http/api/root/server"
	"tty.fail/mrus/korda/http/api/root/sessions"
	"tty.fail/mrus/korda/http/api/root/sharedlinks"
	"tty.fail/mrus/korda/http/api/root/stacks"
	"tty.fail/mrus/korda/http/api/root/sync"
	"tty.fail/mrus/korda/http/api/root/systemconfig"
	"tty.fail/mrus/korda/http/api/root/tags"
	"tty.fail/mrus/korda/http/api/root/timeline"
	"tty.fail/mrus/korda/http/api/root/trash"
	"tty.fail/mrus/korda/http/api/root/users"
	"tty.fail/mrus/korda/http/api/root/view"
	"tty.fail/mrus/korda/http/api/root/workflows"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
)

type Route struct {
	route.RouteController
}

func New(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	r := new(Route)

	r.Runtime = rt
	r.Router = router
	r.Path = route.For("API").AsURL()
	r.Env = route.NewEnv()

	serverRoute, err := server.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, serverRoute)

	authRoute, err := auth.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, authRoute)

	sessionsRoute, err := sessions.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, sessionsRoute)

	sharedLinksRoute, err := sharedlinks.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, sharedLinksRoute)

	usersRoute, err := users.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, usersRoute)

	adminRoute, err := admin.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, adminRoute)

	apikeysRoute, err := apikeys.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, apikeysRoute)

	assetsRoute, err := assets.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, assetsRoute)

	assetFilesRoute, err := assetfiles.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, assetFilesRoute)

	activitiesRoute, err := activities.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, activitiesRoute)

	albumsRoute, err := albums.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, albumsRoute)

	memoriesRoute, err := memories.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, memoriesRoute)

	for _, mount := range []func(*runtime.Runtime, fiber.Router) (*systemconfig.Route, error){
		systemconfig.NewSystem, systemconfig.NewUser, systemconfig.NewPublic, systemconfig.NewMetadata,
	} {
		configRoute, err := mount(r.Runtime, r.Router)
		if err != nil {
			return nil, err
		}
		r.Routes = append(r.Routes, configRoute)
	}

	oauthRoute, err := oauth.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, oauthRoute)

	librariesRoute, err := libraries.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, librariesRoute)

	mapRoute, err := geomap.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, mapRoute)

	duplicatesRoute, err := duplicates.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, duplicatesRoute)

	peopleRoute, err := people.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, peopleRoute)

	facesRoute, err := faces.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, facesRoute)

	queuesRoute, err := queues.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, queuesRoute)

	jobsRoute, err := jobs.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, jobsRoute)

	notificationsRoute, err := notifications.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, notificationsRoute)

	partnersRoute, err := partners.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, partnersRoute)

	stacksRoute, err := stacks.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, stacksRoute)

	tagsRoute, err := tags.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, tagsRoute)

	searchRoute, err := search.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, searchRoute)

	viewRoute, err := view.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, viewRoute)

	downloadRoute, err := download.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, downloadRoute)

	timelineRoute, err := timeline.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, timelineRoute)

	trashRoute, err := trash.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, trashRoute)

	syncRoute, err := sync.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, syncRoute)

	clusterGroupsRoute, err := clustergroups.New(r.Runtime, r.Router)
	if err != nil {
		return nil, err
	}
	r.Routes = append(r.Routes, clusterGroupsRoute)

	for _, mount := range []func(*runtime.Runtime, fiber.Router) (*workflows.Route, error){
		workflows.NewPlugins, workflows.New,
	} {
		workflowsRoute, err := mount(r.Runtime, r.Router)
		if err != nil {
			return nil, err
		}
		r.Routes = append(r.Routes, workflowsRoute)
	}

	return r, nil
}
