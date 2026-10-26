package systemconfig

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/models/systemmetadata"
	"xn--gckvb8fzb.com/glides/runtime"
)

var (
	//go:embed immich.defaults.json
	immichDefaults []byte

	//go:embed immich.order.json
	DefaultsOrder []byte
)

var UserLeaves = []string{
	"ffmpeg.realtime.enabled", "ffmpeg.realtime.videoCodecs", "ffmpeg.realtime.resolutions",
	"machineLearning.enabled", "machineLearning.clip.enabled", "machineLearning.duplicateDetection.enabled",
	"machineLearning.facialRecognition.enabled", "machineLearning.facialRecognition.minFaces",
	"machineLearning.ocr.enabled", "map.enabled", "map.lightStyle", "map.darkStyle", "reverseGeocoding.enabled",
	"oauth.autoLaunch", "oauth.buttonText", "oauth.enabled", "passwordLogin.enabled", "image.thumbnail.size",
	"image.preview.size", "image.fullsize.enabled", "trash.enabled", "trash.days", "theme.customCss",
	"server.externalDomain", "server.loginPageMessage", "server.publicUsers", "user.deleteDelay",
}

var PublicLeaves = []string{
	"oauth.autoLaunch", "oauth.buttonText", "oauth.enabled", "passwordLogin.enabled", "theme.customCss",
	"server.loginPageMessage",
}

var logLevels = map[string]LogLevel{"debug": "debug", "info": "log", "warn": "warn", "error": "error"}

var defaultPorts = map[string]string{"ftp": "21", "http": "80", "https": "443", "ws": "80", "wss": "443"}

func Origin(domain string) string {
	if domain == "" {
		return ""
	}
	u, err := url.Parse(domain)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return domain
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port := u.Port(); port != "" && port != defaultPorts[scheme] {
		host += ":" + port
	}
	if password, ok := u.User.Password(); ok && password != "" && u.User.Username() != "" {
		return scheme + "://" + u.User.String() + "@" + host
	}
	if _, special := defaultPorts[scheme]; !special {
		return "null"
	}

	return scheme + "://" + host
}

func base() (*Config, error) {
	cfg := new(Config)
	dec := json.NewDecoder(bytes.NewReader(immichDefaults))
	dec.DisallowUnknownFields()

	return cfg, dec.Decode(cfg)
}

func apply(cfg *Config, media gh.MediaConfig, server gh.ServerConfig, level string) {
	cfg.Backup.Database.Enabled = false

	cfg.FFmpeg.Transcode = TranscodePolicy(media.Transcode)
	cfg.FFmpeg.TargetResolution = media.TargetResolution

	cfg.IntegrityChecks.MissingFiles.Enabled = false
	cfg.IntegrityChecks.UntrackedFiles.Enabled = false
	cfg.IntegrityChecks.ChecksumFiles.Enabled = false

	concurrency := int64(media.Concurrency)
	cfg.Job.ThumbnailGeneration.Concurrency = concurrency
	cfg.Job.MetadataExtraction.Concurrency = concurrency
	cfg.Job.VideoConversion.Concurrency = concurrency
	cfg.Job.Editor.Concurrency = concurrency

	if l, ok := logLevels[level]; ok {
		cfg.Logging.Level = l
	}

	cfg.MachineLearning.Enabled = false
	cfg.MachineLearning.Clip.Enabled = false
	cfg.MachineLearning.DuplicateDetection.Enabled = false
	cfg.MachineLearning.FacialRecognition.Enabled = false
	cfg.MachineLearning.OCR.Enabled = false

	cfg.Map.Enabled = false
	cfg.ReverseGeocoding.Enabled = false

	cfg.Image.Thumbnail.Format = "jpeg"
	cfg.Image.Thumbnail.Quality = int64(media.ThumbnailQuality)
	cfg.Image.Thumbnail.Size = int64(media.ThumbnailSize)
	cfg.Image.Preview.Quality = int64(media.PreviewQuality)
	cfg.Image.Preview.Size = int64(media.PreviewSize)
	cfg.Image.Colorspace = "srgb"

	cfg.NewVersionCheck.Enabled = false
	cfg.NightlyTasks.ClusterNewFaces = false

	cfg.Trash.Days = int64(server.TrashDays)
	cfg.Library.Scan.Enabled = false

	cfg.Server.ExternalDomain = Origin(server.ExternalDomain)
	cfg.Server.LoginPageMessage = server.LoginPageMessage
	cfg.Server.PublicUsers = server.PublicUsers
	cfg.User.DeleteDelay = int64(server.UserDeleteDelay)
}

func Effective(rt *runtime.Runtime) (*Config, error) {
	cfg, err := base()
	if err != nil {
		return nil, err
	}
	media, err := gh.MediaConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}
	server, err := gh.ServerConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}

	apply(cfg, media, server, rt.Config().LoggingLevel())

	return cfg, nil
}

func Defaults() (*Config, error) {
	cfg, err := base()
	if err != nil {
		return nil, err
	}

	apply(cfg, gh.NewMediaConfig(), gh.NewServerConfig(), "info")

	return cfg, nil
}

func Update(in *Config) error {
	return errs.ErrConfigFile
}

//go:embed immich.storage-template.json
var StorageTemplateOptions []byte

type OnboardingInput struct {
	IsOnboarded bool `json:"isOnboarded"`
}

func Onboarding(ctx context.Context, rt *runtime.Runtime) (bool, error) {
	raw, err := gh.Repositories(rt).SystemMetadata.Get(ctx, systemmetadata.AdminOnboarding)
	if errors.Is(err, errs.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	var value systemmetadata.AdminOnboardingValue
	if err = json.Unmarshal([]byte(raw), &value); err != nil {
		return false, err
	}

	return value.IsOnboarded, nil
}

func SetOnboarding(ctx context.Context, rt *runtime.Runtime, in *OnboardingInput) error {
	encoded, err := json.Marshal(systemmetadata.AdminOnboardingValue{IsOnboarded: in.IsOnboarded})
	if err != nil {
		return err
	}

	return gh.Repositories(rt).SystemMetadata.Set(ctx, systemmetadata.AdminOnboarding, string(encoded))
}
