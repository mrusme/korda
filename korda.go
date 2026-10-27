package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/helpers/optional"
	"tty.fail/mrus/korda/http/routes"
	"tty.fail/mrus/korda/http/web"
	logicadminusers "tty.fail/mrus/korda/logic/root/admin/users"
	logicassets "tty.fail/mrus/korda/logic/root/assets"
	logicmemories "tty.fail/mrus/korda/logic/root/memories"
	logicserver "tty.fail/mrus/korda/logic/root/server"
	logicsessions "tty.fail/mrus/korda/logic/root/sessions"
	logicsync "tty.fail/mrus/korda/logic/root/sync"
	logicusers "tty.fail/mrus/korda/logic/root/users"
	"tty.fail/mrus/korda/migrations"
	"tty.fail/mrus/korda/services/events"
	"tty.fail/mrus/korda/services/jobs"
	"tty.fail/mrus/korda/services/media"
	"tty.fail/mrus/korda/services/mediaroot"
	"tty.fail/mrus/korda/services/metadata"
	"tty.fail/mrus/korda/services/repositories"
	"tty.fail/mrus/korda/services/websocket"
	"tty.fail/mrus/korda/worker/targets"
	"xn--gckvb8fzb.com/glides/cron"
	glideshttp "xn--gckvb8fzb.com/glides/http"
	"xn--gckvb8fzb.com/glides/http/route"
	"xn--gckvb8fzb.com/glides/runtime"
	servicescron "xn--gckvb8fzb.com/glides/services/cron"
	"xn--gckvb8fzb.com/glides/services/database"
	"xn--gckvb8fzb.com/glides/services/database/sqlite"
	glidesdispatch "xn--gckvb8fzb.com/glides/services/dispatch"
	"xn--gckvb8fzb.com/glides/worker"
)

var (
	Version string
	Commit  string
	Date    string
)

const (
	databaseConnections = 4
	nightlySpec         = "0 0 * * *"
	userDeleteCheckID   = "user-delete-check"
	sessionCleanupID    = "session-cleanup"
	assetDeleteCheckID  = "asset-delete-check"
	userSyncUsageID     = "user-sync-usage"
	missingThumbsID     = "missing-thumbnails"
	memoryCleanupID     = "memory-cleanup"
	memoryGenerateID    = "memory-generate"
	auditCleanupID      = "audit-table-cleanup"
	adminStorageLabel   = "admin"
)

var (
	flagCfgstr     string
	flagVersion    bool
	flagReset      string
	flagCreateUser string
	flagMigrate    bool
)

func init() {
	flag.StringVar(&flagCfgstr, "c", "file:///etc/korda.toml", "configuration string")
	flag.BoolVar(&flagVersion, "v", false, "Print version information and exit")
	flag.StringVar(&flagReset, "reset", "", "Clear the whole database and exit (requires the current time as HH:MM (24h) confirmation, e.g. --reset 10:42)")
	flag.StringVar(&flagCreateUser, "create-user", "", `Create a user from a JSON object and exit, e.g. --create-user '{"email":"admin@example.com","name":"Admin","password":"mypassword","admin":true}'`)
	flag.BoolVar(&flagMigrate, "migrate", false, "Run the database migrations and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Use: %s [-opts]\n\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	var rt *runtime.Runtime
	var err error

	flag.Parse()

	if flagVersion {
		fmt.Printf("korda %s\nCommit: %s\nBuild date: %s\nImmich API: %s\n",
			Version,
			Commit,
			Date,
			logicserver.ImmichVersion,
		)
		os.Exit(0)
	}

	if rt, err = newRuntime(flagCfgstr, sqlite.Options{MaxOpenConns: databaseConnections}); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	if flagReset != "" {
		if err = runtime.ValidateResetTime(flagReset, time.Now()); err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		if err = database.From(rt).Reset(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if flagMigrate {
		if err = migrate(rt); err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if flagCreateUser != "" {
		if err = createUser(rt, flagCreateUser); err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if _, err = glidesdispatch.Register(rt); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	err = targets.Check(rt.Config())
	rt.NilOrDie(err)

	redisCfg, err := rt.Config().Redis()
	rt.NilOrDie(err)
	dispatcher := jobs.NewDispatcher(glidesdispatch.From(rt), redisCfg)
	defer dispatcher.Close()

	err = prepare(rt, dispatcher)
	rt.NilOrDie(err)

	err = rt.Startup()
	rt.NilOrDie(err)

	route.Use(routes.Routes, routes.DefaultTitle)

	err = logicserver.RecordVersion(context.Background(), rt)
	rt.NilOrDie(err)

	webcfg, err := gh.WebConfigFrom(rt.Config())
	rt.NilOrDie(err)
	apicfg, err := gh.APIConfigFrom(rt.Config())
	rt.NilOrDie(err)
	mediacfg, err := gh.MediaConfigFrom(rt.Config())
	rt.NilOrDie(err)

	wrk, err := worker.New(rt, targets.Register(rt))
	rt.NilOrDie(err)
	wrk.SetConcurrency(mediacfg.Concurrency + 1)
	wrk.SetQueues(jobs.Queues())

	webiface, err := web.New(rt, webcfg, apicfg)
	rt.NilOrDie(err)
	websrv, err := glideshttp.New(rt, webiface)
	rt.NilOrDie(err)

	err = registerCronFunctions(rt)
	rt.NilOrDie(err)
	crn, err := cron.New(rt)
	rt.NilOrDie(err)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err = rt.Serve(ctx, wrk, websrv, crn); err != nil {
		rt.Exit(1)
	}

	rt.Exit(0)
}

func newRuntime(cfgstr string, dbopts sqlite.Options) (rt *runtime.Runtime, err error) {
	if rt, err = runtime.New(runtime.Opts{
		Cfgstr:  cfgstr,
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	}); err != nil {
		return nil, err
	}

	dbopts.Migrations = &migrations.FS
	dbopts.MigrationsPath = migrations.Path
	if _, err = sqlite.Register(rt, dbopts); err != nil {
		return nil, err
	}
	if _, err = servicescron.Register(rt); err != nil {
		return nil, err
	}

	return rt, nil
}

func prepare(rt *runtime.Runtime, queue jobs.Enqueuer) (err error) {
	mediacfg, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return err
	}

	rt.Debug("new", "repositories")
	repos, err := repositories.New(database.From(rt))
	if err != nil {
		return err
	}
	rt.AddService("repositories", repos)

	rt.Debug("new", "mediaroot")
	root, err := mediaroot.New(mediacfg.Location)
	if err != nil {
		return err
	}
	rt.AddService("mediaroot", root)

	rt.Debug("new", "media")
	med, err := media.New(root, mediacfg.ThumbnailSize, mediacfg.ThumbnailQuality)
	if err != nil {
		return err
	}
	if ff := media.DetectFFmpeg(context.Background(), mediacfg.FFmpeg); ff != nil {
		med.SetFFmpeg(ff)
		rt.Info("media", "ffmpeg", "path", ff.Path, "encoder", ff.Encoder, "hevc", ff.HEVC, "tone mapping", ff.ToneMap)
		if !ff.HEVC {
			rt.Warn("media", "ffmpeg", "no HEVC decoder", "HEVC videos, such as those of iPhones, get no thumbnails and are not transcoded")
		}
	} else {
		rt.Warn("media", "ffmpeg", "not available", "videos get no thumbnails and are not transcoded")
	}
	rt.Info("media", "build tags", "heic", media.HEIC, "tzf", metadata.GPSZones)
	rt.AddService("media", med)

	rt.Debug("new", "jobs")
	jbs, err := jobs.New(queue)
	if err != nil {
		return err
	}
	rt.AddService("jobs", jbs)

	rt.Debug("new", "events")
	bus, err := events.New()
	if err != nil {
		return err
	}
	rt.AddService("events", bus)

	rt.Debug("new", "websocket")
	hub, err := websocket.New()
	if err != nil {
		return err
	}
	rt.AddService("websocket", hub)

	rt.OnStartup(repos.Startup, root.Startup, med.Startup, jbs.Startup, hub.Startup)
	rt.OnShutdown(repos.Shutdown, root.Shutdown, med.Shutdown, jbs.Shutdown, hub.Shutdown)

	return nil
}

func registerCronFunctions(rt *runtime.Runtime) (err error) {
	crn := servicescron.From(rt)

	if err = crn.Register(userDeleteCheckID, nightlySpec, func(ctx context.Context) error {
		return logicadminusers.QueueDueDeletions(ctx, rt)
	}); err != nil {
		return err
	}

	if err = crn.Register(sessionCleanupID, nightlySpec, func(ctx context.Context) error {
		return logicsessions.Cleanup(ctx, rt)
	}); err != nil {
		return err
	}

	if err = crn.Register(assetDeleteCheckID, nightlySpec, func(ctx context.Context) error {
		servercfg, err := gh.ServerConfigFrom(rt.Config())
		if err != nil {
			return err
		}
		return logicassets.QueueExpired(ctx, rt, servercfg.TrashDays)
	}); err != nil {
		return err
	}

	if err = crn.Register(userSyncUsageID, nightlySpec, func(ctx context.Context) error {
		return gh.Repositories(rt).User.SyncUsage(ctx, time.Now().UTC())
	}); err != nil {
		return err
	}

	if err = crn.Register(missingThumbsID, nightlySpec, func(ctx context.Context) error {
		return logicassets.QueueMissingThumbnails(ctx, rt)
	}); err != nil {
		return err
	}

	if err = crn.Register(memoryCleanupID, nightlySpec, func(ctx context.Context) error {
		return logicmemories.Cleanup(ctx, rt)
	}); err != nil {
		return err
	}

	if err = crn.Register(memoryGenerateID, nightlySpec, func(ctx context.Context) error {
		return logicmemories.Generate(ctx, rt)
	}); err != nil {
		return err
	}

	return crn.Register(auditCleanupID, nightlySpec, func(ctx context.Context) error {
		return logicsync.CleanupAudit(ctx, rt)
	})
}

func migrate(rt *runtime.Runtime) error {
	if err := database.From(rt).Startup(); err != nil {
		return err
	}

	return database.From(rt).Shutdown()
}

type createUserInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
}

func createUser(rt *runtime.Runtime, jsonStr string) error {
	in := new(createUserInput)
	if err := json.Unmarshal([]byte(jsonStr), in); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrCreateUserInvalid, err)
	}
	if in.Email == "" || in.Name == "" || in.Password == "" {
		return fmt.Errorf("%w: email, name and password are required", errs.ErrCreateUserInvalid)
	}

	if err := prepare(rt, nil); err != nil {
		return err
	}
	if err := database.From(rt).Startup(); err != nil {
		return err
	}
	defer database.From(rt).Shutdown()

	if err := gh.Repositories(rt).Startup(); err != nil {
		return err
	}

	ctx := context.Background()
	create := logicusers.CreateInput{
		Email:    in.Email,
		Name:     in.Name,
		Password: in.Password,
		IsAdmin:  in.Admin,
	}
	if in.Admin {
		hasAdmin, err := gh.Repositories(rt).User.HasAdmin(ctx)
		if err != nil {
			return err
		}
		if !hasAdmin {
			create.StorageLabel = optional.Of(adminStorageLabel)
		}
	}

	u, err := logicusers.Create(ctx, rt, create)
	if err != nil {
		return err
	}

	rt.Info("create-user", "ok", "id", u.ID.String(), "email", u.Email, "admin", u.IsAdmin)
	return nil
}
