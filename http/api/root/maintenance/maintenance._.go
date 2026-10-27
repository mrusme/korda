package maintenance

import (
	"errors"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"tty.fail/mrus/korda/errs"
	"tty.fail/mrus/korda/http/api/request"
	logicauth "tty.fail/mrus/korda/logic/root/auth"
	logicmaintenance "tty.fail/mrus/korda/logic/root/maintenance"
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

func NewMaintenance(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APIAdminMaintenance", func(r *Route, base fiber.Router) {
		base.Get("/status", r.GetMaintenanceStatus).Name("getMaintenanceStatus")
		base.Get("/detect-install", r.DetectPriorInstall).Name("detectPriorInstall")
		base.Post("/login", r.MaintenanceLogin).Name("maintenanceLogin")
		base.Post("", r.SetMaintenanceMode).Name("setMaintenanceMode")
	}), nil
}

func NewIntegrity(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APIAdminIntegrity", func(r *Route, base fiber.Router) {
		base.Get("/summary", r.GetIntegrityReportSummary).Name("getIntegrityReportSummary")
		base.Get("/report", r.GetIntegrityReport).Name("getIntegrityReport")
		base.Get("/report/:id/file", r.GetIntegrityReportFile).Name("getIntegrityReportFile")
		base.Delete("/report/:id", r.DeleteIntegrityReport).Name("deleteIntegrityReport")
		base.Get("/report/:type/csv", r.GetIntegrityReportCsv).Name("getIntegrityReportCsv")
	}), nil
}

func NewBackups(rt *runtime.Runtime, router fiber.Router) (*Route, error) {
	return mount(rt, router, "APIAdminDatabaseBackups", func(r *Route, base fiber.Router) {
		base.Get("", r.ListDatabaseBackups).Name("listDatabaseBackups")
		base.Delete("", r.DeleteDatabaseBackup).Name("deleteDatabaseBackup")
		base.Post("/start-restore", r.StartDatabaseRestoreFlow).Name("startDatabaseRestoreFlow")
		base.Post("/upload", r.UploadDatabaseBackup).Name("uploadDatabaseBackup")
		base.Get("/:filename", r.DownloadDatabaseBackup).Name("downloadDatabaseBackup")
	}), nil
}

type status struct {
	Active bool   `json:"active"`
	Action string `json:"action"`
}

type storageFolder struct {
	Folder   string `json:"folder"`
	Readable bool   `json:"readable"`
	Writable bool   `json:"writable"`
	Files    int    `json:"files"`
}

type detectInstall struct {
	Storage []storageFolder `json:"storage"`
}

type summary struct {
	UntrackedFile    int `json:"untracked_file"`
	MissingFile      int `json:"missing_file"`
	ChecksumMismatch int `json:"checksum_mismatch"`
}

type report struct {
	Items []struct{} `json:"items"`
}

type backup struct {
	Filename string `json:"filename"`
	Filesize int64  `json:"filesize"`
	Timezone string `json:"timezone"`
}

type backups struct {
	Backups []backup `json:"backups"`
}

func admin(p permission.Permission) logicauth.Requirement {
	return logicauth.Requirement{Permission: p, Admin: true}
}

func (r *Route) GetMaintenanceStatus(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getMaintenanceStatus")
	return req.Respond(fiber.StatusOK, status{Action: "end"})
}

func (r *Route) DetectPriorInstall(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "detectPriorInstall")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	out := detectInstall{Storage: []storageFolder{}}
	for _, f := range logicmaintenance.DetectInstall(r.Runtime) {
		out.Storage = append(out.Storage, storageFolder{Folder: f.Folder, Readable: f.Readable, Writable: f.Writable, Files: f.Files})
	}

	return req.Respond(fiber.StatusOK, out)
}

func (r *Route) MaintenanceLogin(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "maintenanceLogin")

	var in logicmaintenance.LoginInput
	if err := req.Body(&in); err != nil {
		return err
	}

	return logicmaintenance.Login(&in)
}

func (r *Route) SetMaintenanceMode(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "setMaintenanceMode")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	var in logicmaintenance.SetModeInput
	if err := req.Body(&in); err != nil {
		return err
	}
	if err := logicmaintenance.SetMode(&in); err != nil {
		return err
	}

	return req.Empty(fiber.StatusCreated)
}

func (r *Route) GetIntegrityReportSummary(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getIntegrityReportSummary")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	return req.Respond(fiber.StatusOK, summary{})
}

func (r *Route) GetIntegrityReport(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getIntegrityReport")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	var in logicmaintenance.ReportInput
	if err := req.Query(&in); err != nil {
		return err
	}
	if in.Cursor != nil && *in.Cursor != "" && !validUUID(*in.Cursor) {
		return errs.ErrInternal
	}

	return req.Respond(fiber.StatusOK, report{Items: []struct{}{}})
}

func (r *Route) GetIntegrityReportFile(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getIntegrityReportFile")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	var p logicmaintenance.ReportIDParams
	if err := req.Params(&p, "id"); err != nil {
		return err
	}

	return errs.ErrNotFound
}

func (r *Route) DeleteIntegrityReport(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteIntegrityReport")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	var p logicmaintenance.ReportIDParams
	if err := req.Params(&p, "id"); err != nil {
		return err
	}

	return errs.ErrInternal
}

func (r *Route) GetIntegrityReportCsv(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "getIntegrityReportCsv")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	var p logicmaintenance.ReportTypeParams
	if err := req.Params(&p, "type"); err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, "text/csv; charset=utf-8")
	c.Set(fiber.HeaderCacheControl, "private, no-cache, no-transform")
	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+strconv.FormatInt(time.Now().UnixMilli(), 10)+"-"+string(p.Type)+`.csv"`)

	return c.Status(fiber.StatusOK).SendString("id,type,assetId,fileAssetId,path\n")
}

func (r *Route) ListDatabaseBackups(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "listDatabaseBackups")
	if err := req.Require(admin(permission.Maintenance)); err != nil {
		return err
	}

	list, err := logicmaintenance.ListBackups(r.Runtime)
	if err != nil {
		return err
	}
	out := backups{Backups: make([]backup, 0, len(list))}
	for _, b := range list {
		out.Backups = append(out.Backups, backup{Filename: b.Filename, Filesize: b.Filesize, Timezone: b.Timezone})
	}

	return req.Respond(fiber.StatusOK, out)
}

func (r *Route) DownloadDatabaseBackup(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "downloadDatabaseBackup")
	if err := req.Require(admin(permission.BackupDownload)); err != nil {
		return err
	}

	var p logicmaintenance.FilenameParams
	if err := req.Params(&p, "filename"); err != nil {
		return err
	}
	path, err := logicmaintenance.BackupPath(r.Runtime, p.Filename)
	if err != nil {
		return errs.ErrNotFound
	}
	contentType := "application/sql"
	if strings.HasSuffix(p.Filename, ".gz") {
		contentType = "application/gzip"
	}

	return req.SendMediaWith(path, contentType, p.Filename, request.CacheUncached)
}

func (r *Route) DeleteDatabaseBackup(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "deleteDatabaseBackup")
	if err := req.Require(admin(permission.BackupDelete)); err != nil {
		return err
	}

	var in logicmaintenance.DeleteBackupsInput
	if err := req.Body(&in); err != nil {
		return err
	}

	if err := logicmaintenance.DeleteBackups(r.Runtime, &in); err != nil {
		return err
	}

	return req.Empty(fiber.StatusOK)
}

func (r *Route) StartDatabaseRestoreFlow(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "startDatabaseRestoreFlow")
	if err := req.RequireSetup(); err != nil {
		return err
	}

	return errs.UnsupportedRestore
}

func (r *Route) UploadDatabaseBackup(c fiber.Ctx) error {
	req := request.New(r.Runtime, c, "uploadDatabaseBackup")
	if err := req.Require(admin(permission.BackupUpload)); err != nil {
		return err
	}

	err := req.File("file", func(filename string) error {
		return nil
	}, func(filename string, src io.Reader) error {
		if !logicmaintenance.ValidBackupName(filepath.Base(filename)) {
			if _, err := io.Copy(io.Discard, src); err != nil {
				return err
			}
			return errs.ErrInvalidBackupName
		}
		return logicmaintenance.UploadBackup(r.Runtime, filename, src)
	})
	if errors.Is(err, request.ErrNoFile) {
		return errs.ErrInternal
	}
	if err != nil {
		return err
	}

	return req.Empty(fiber.StatusCreated)
}

func validUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
