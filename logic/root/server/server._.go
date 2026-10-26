package server

import (
	"context"
	"encoding/json"
	"errors"

	"tty.fail/mrus/korda/errs"
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/logic/root/systemconfig"
	"tty.fail/mrus/korda/models/systemmetadata"
	"tty.fail/mrus/korda/models/versionhistory"
	"xn--gckvb8fzb.com/glides/runtime"
)

const (
	ImmichMajor   = 3
	ImmichMinor   = 3
	ImmichPatch   = 3
	ImmichVersion = "3.3.3"

	RepositoryURL = "https://tty.fail/mrus/korda"

	mapLightStyle   = "https://tiles.immich.cloud/v1/style/light.json"
	mapDarkStyle    = "https://tiles.immich.cloud/v1/style/dark.json"
	oauthButtonText = "Login with OAuth"
	minFaces        = 3
)

type Features struct {
	SmartSearch         bool
	FacialRecognition   bool
	DuplicateDetection  bool
	Map                 bool
	ReverseGeocoding    bool
	ImportFaces         bool
	Sidecar             bool
	Search              bool
	Trash               bool
	OAuth               bool
	OAuthAutoLaunch     bool
	OCR                 bool
	PasswordLogin       bool
	ConfigFile          bool
	Email               bool
	RealtimeTranscoding bool
}

func GetFeatures() Features {
	return Features{
		Sidecar:       true,
		Search:        true,
		Trash:         true,
		PasswordLogin: true,
		ConfigFile:    true,
	}
}

type Config struct {
	LoginPageMessage          string
	TrashDays                 int
	UserDeleteDelay           int
	OAuthButtonText           string
	OAuthAccountManagementURL string
	IsInitialized             bool
	IsOnboarded               bool
	ExternalDomain            string
	PublicUsers               bool
	MapDarkStyleURL           string
	MapLightStyleURL          string
	MaintenanceMode           bool
	MinFaces                  int
}

func GetConfig(ctx context.Context, rt *runtime.Runtime) (*Config, error) {
	cfg, err := gh.ServerConfigFrom(rt.Config())
	if err != nil {
		return nil, err
	}

	repos := gh.Repositories(rt)

	initialized, err := repos.User.HasAdmin(ctx)
	if err != nil {
		return nil, err
	}

	onboarded := false
	raw, err := repos.SystemMetadata.Get(ctx, systemmetadata.AdminOnboarding)
	switch {
	case errors.Is(err, errs.ErrNoRows):
	case err != nil:
		return nil, err
	default:
		var value systemmetadata.AdminOnboardingValue
		if err = json.Unmarshal([]byte(raw), &value); err != nil {
			return nil, err
		}
		onboarded = value.IsOnboarded
	}

	return &Config{
		LoginPageMessage:          cfg.LoginPageMessage,
		TrashDays:                 cfg.TrashDays,
		UserDeleteDelay:           cfg.UserDeleteDelay,
		OAuthButtonText:           oauthButtonText,
		OAuthAccountManagementURL: "",
		IsInitialized:             initialized,
		IsOnboarded:               onboarded,
		ExternalDomain:            systemconfig.Origin(cfg.ExternalDomain),
		PublicUsers:               cfg.PublicUsers,
		MapDarkStyleURL:           mapDarkStyle,
		MapLightStyleURL:          mapLightStyle,
		MaintenanceMode:           false,
		MinFaces:                  minFaces,
	}, nil
}

func VersionHistory(ctx context.Context, rt *runtime.Runtime) ([]versionhistory.VersionHistory, error) {
	return gh.Repositories(rt).VersionHistory.List(ctx)
}

func RecordVersion(ctx context.Context, rt *runtime.Runtime) error {
	version, _, _, _ := rt.GetBuild()
	if version == "" {
		version = "dev"
	}

	repos := gh.Repositories(rt)
	latest, err := repos.VersionHistory.Latest(ctx)
	if err != nil && !errors.Is(err, errs.ErrNoRows) {
		return err
	}
	if latest != nil && latest.Version == version {
		return nil
	}

	return repos.VersionHistory.Create(ctx, version, now())
}
