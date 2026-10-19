package helpers

import (
	"xn--gckvb8fzb.com/glides/services/config"
)

const (
	WebDefaultBindIP         string = "0.0.0.0"
	WebDefaultPort           int    = 2283
	WebDefaultBodyLimit      int    = 10 * 1024 * 1024 * 1024
	WebDefaultConcurrency    int    = 256 * 1024
	WebDefaultRequestTimeout int    = 30
	APIDefaultRateLimitAuth  int    = 10
	MediaDefaultLocation     string = "./media"
	MediaDefaultConcurrency  int    = 1
	MediaDefaultThumbSize    int    = 250
	MediaDefaultThumbQuality int    = 80
	ServerDefaultTrashDays   int    = 30
	ServerDefaultDeleteDelay int    = 7
)

type WebConfig struct {
	BindIP             string   `koanf:"BindIP"`
	Port               int      `koanf:"Port"`
	BodyLimit          int      `koanf:"BodyLimit"`
	Concurrency        int      `koanf:"Concurrency"`
	ProxyHeader        string   `koanf:"ProxyHeader"`
	EnableIPValidation bool     `koanf:"EnableIPValidation"`
	TrustProxy         bool     `koanf:"TrustProxy"`
	TrustLoopback      bool     `koanf:"TrustLoopback"`
	TrustProxies       []string `koanf:"TrustProxies"`
	ReduceMemoryUsage  bool     `koanf:"ReduceMemoryUsage"`
	ServerHeader       string   `koanf:"ServerHeader"`
	RequestTimeout     int      `koanf:"RequestTimeout"`
}

type APIConfig struct {
	RateLimitAuth int      `koanf:"RateLimitAuth"`
	CORSOrigins   []string `koanf:"CORSOrigins"`
}

type MediaConfig struct {
	Location         string `koanf:"Location"`
	Concurrency      int    `koanf:"Concurrency"`
	ThumbnailSize    int    `koanf:"ThumbnailSize"`
	ThumbnailQuality int    `koanf:"ThumbnailQuality"`
}

type ServerConfig struct {
	ExternalDomain   string `koanf:"ExternalDomain"`
	LoginPageMessage string `koanf:"LoginPageMessage"`
	PublicUsers      bool   `koanf:"PublicUsers"`
	TrashDays        int    `koanf:"TrashDays"`
	UserDeleteDelay  int    `koanf:"UserDeleteDelay"`
}

func NewWebConfig() WebConfig {
	return WebConfig{
		BindIP:         WebDefaultBindIP,
		Port:           WebDefaultPort,
		BodyLimit:      WebDefaultBodyLimit,
		Concurrency:    WebDefaultConcurrency,
		TrustLoopback:  true,
		RequestTimeout: WebDefaultRequestTimeout,
	}
}

func NewAPIConfig() APIConfig {
	return APIConfig{
		RateLimitAuth: APIDefaultRateLimitAuth,
	}
}

func NewMediaConfig() MediaConfig {
	return MediaConfig{
		Location:         MediaDefaultLocation,
		Concurrency:      MediaDefaultConcurrency,
		ThumbnailSize:    MediaDefaultThumbSize,
		ThumbnailQuality: MediaDefaultThumbQuality,
	}
}

func NewServerConfig() ServerConfig {
	return ServerConfig{
		PublicUsers:     true,
		TrashDays:       ServerDefaultTrashDays,
		UserDeleteDelay: ServerDefaultDeleteDelay,
	}
}

func WebConfigFrom(cfg *config.Config) (web WebConfig, err error) {
	web = NewWebConfig()
	if err = cfg.Unmarshal("Web", &web); err != nil {
		return NewWebConfig(), err
	}

	return web, nil
}

func APIConfigFrom(cfg *config.Config) (api APIConfig, err error) {
	api = NewAPIConfig()
	if err = cfg.Unmarshal("API", &api); err != nil {
		return NewAPIConfig(), err
	}

	return api, nil
}

func MediaConfigFrom(cfg *config.Config) (media MediaConfig, err error) {
	media = NewMediaConfig()
	if err = cfg.Unmarshal("Media", &media); err != nil {
		return NewMediaConfig(), err
	}

	return media, nil
}

func ServerConfigFrom(cfg *config.Config) (server ServerConfig, err error) {
	server = NewServerConfig()
	if err = cfg.Unmarshal("Server", &server); err != nil {
		return NewServerConfig(), err
	}

	return server, nil
}
