package systemconfig

import "tty.fail/mrus/korda/helpers/optional"

type VideoCodec string

func (VideoCodec) Enum() []string { return []string{"h264", "hevc", "vp9", "av1"} }

type AudioCodec string

func (AudioCodec) Enum() []string { return []string{"mp3", "aac", "opus", "pcm_s16le"} }

type Container string

func (Container) Enum() []string { return []string{"mov", "mp4", "ogg", "webm"} }

type CQMode string

func (CQMode) Enum() []string { return []string{"auto", "cqp", "icq"} }

type TranscodePolicy string

func (TranscodePolicy) Enum() []string {
	return []string{"all", "optimal", "bitrate", "required", "disabled"}
}

type Accel string

func (Accel) Enum() []string { return []string{"nvenc", "qsv", "vaapi", "rkmpp", "disabled"} }

type Tonemap string

func (Tonemap) Enum() []string { return []string{"hable", "mobius", "reinhard", "disabled"} }

type Resolution int64

func (Resolution) NumberEnum() []int64 { return []int64{480, 720, 1080, 1440, 2160} }

type LogLevel string

func (LogLevel) Enum() []string { return []string{"verbose", "debug", "log", "warn", "error", "fatal"} }

type AuthMethod string

func (AuthMethod) Enum() []string { return []string{"client_secret_post", "client_secret_basic"} }

type ImageFormat string

func (ImageFormat) Enum() []string { return []string{"jpeg", "webp"} }

type Colorspace string

func (Colorspace) Enum() []string { return []string{"srgb", "p3"} }

type Channel string

func (Channel) Enum() []string { return []string{"stable", "releaseCandidate"} }

type Cron struct {
	Enabled        bool   `json:"enabled" v:"configbool"`
	CronExpression string `json:"cronExpression" v:"refine=cron"`
}

type Backup struct {
	Database struct {
		Enabled        bool   `json:"enabled" v:"configbool"`
		CronExpression string `json:"cronExpression" v:"refine=cron"`
		KeepLastAmount int64  `json:"keepLastAmount" v:"min=1"`
	} `json:"database"`
}

type Realtime struct {
	Enabled     bool         `json:"enabled" v:"configbool"`
	VideoCodecs []VideoCodec `json:"videoCodecs"`
	Resolutions []Resolution `json:"resolutions"`
}

type FFmpeg struct {
	CRF                 int64           `json:"crf" v:"coerce,min=0,max=51"`
	Threads             int64           `json:"threads" v:"coerce,min=0"`
	Preset              string          `json:"preset"`
	TargetVideoCodec    VideoCodec      `json:"targetVideoCodec"`
	AcceptedVideoCodecs []VideoCodec    `json:"acceptedVideoCodecs"`
	TargetAudioCodec    AudioCodec      `json:"targetAudioCodec"`
	AcceptedAudioCodecs []AudioCodec    `json:"acceptedAudioCodecs"`
	AcceptedContainers  []Container     `json:"acceptedContainers"`
	TargetResolution    string          `json:"targetResolution"`
	MaxBitrate          string          `json:"maxBitrate"`
	BFrames             int64           `json:"bframes" v:"coerce,min=-1,max=16"`
	Refs                int64           `json:"refs" v:"coerce,min=0,max=6"`
	GOPSize             int64           `json:"gopSize" v:"coerce,min=0"`
	TemporalAQ          bool            `json:"temporalAQ" v:"configbool"`
	CQMode              CQMode          `json:"cqMode"`
	TwoPass             bool            `json:"twoPass" v:"configbool"`
	PreferredHwDevice   string          `json:"preferredHwDevice"`
	Transcode           TranscodePolicy `json:"transcode"`
	Accel               Accel           `json:"accel"`
	AccelDecode         bool            `json:"accelDecode" v:"configbool"`
	Tonemap             Tonemap         `json:"tonemap"`
	Realtime            Realtime        `json:"realtime"`
}

type IntegrityCheck struct {
	Enabled        bool   `json:"enabled"`
	CronExpression string `json:"cronExpression" v:"refine=cron"`
}

type IntegrityChecks struct {
	MissingFiles   IntegrityCheck `json:"missingFiles"`
	UntrackedFiles IntegrityCheck `json:"untrackedFiles"`
	ChecksumFiles  struct {
		Enabled         bool    `json:"enabled"`
		CronExpression  string  `json:"cronExpression" v:"refine=cron"`
		TimeLimit       int64   `json:"timeLimit" v:"min=0"`
		PercentageLimit float64 `json:"percentageLimit" v:"min=0,max=1"`
	} `json:"checksumFiles"`
}

type Concurrency struct {
	Concurrency int64 `json:"concurrency" v:"min=1"`
}

type Jobs struct {
	ThumbnailGeneration Concurrency `json:"thumbnailGeneration"`
	MetadataExtraction  Concurrency `json:"metadataExtraction"`
	VideoConversion     Concurrency `json:"videoConversion"`
	FaceDetection       Concurrency `json:"faceDetection"`
	SmartSearch         Concurrency `json:"smartSearch"`
	BackgroundTask      Concurrency `json:"backgroundTask"`
	Migration           Concurrency `json:"migration"`
	Search              Concurrency `json:"search"`
	Sidecar             Concurrency `json:"sidecar"`
	Library             Concurrency `json:"library"`
	Notifications       Concurrency `json:"notifications"`
	OCR                 Concurrency `json:"ocr"`
	Workflow            Concurrency `json:"workflow"`
	Editor              Concurrency `json:"editor"`
	IntegrityCheck      Concurrency `json:"integrityCheck"`
}

type Logging struct {
	Enabled bool     `json:"enabled" v:"configbool"`
	Level   LogLevel `json:"level"`
}

type MachineLearning struct {
	Enabled            bool     `json:"enabled" v:"configbool"`
	URLs               []string `json:"urls" v:"min=1"`
	AvailabilityChecks struct {
		Enabled  bool  `json:"enabled" v:"configbool"`
		Timeout  int64 `json:"timeout"`
		Interval int64 `json:"interval"`
	} `json:"availabilityChecks"`
	Clip struct {
		Enabled   bool   `json:"enabled"`
		ModelName string `json:"modelName"`
	} `json:"clip"`
	DuplicateDetection struct {
		Enabled     bool    `json:"enabled"`
		MaxDistance float64 `json:"maxDistance" v:"min=0.001,max=0.1"`
	} `json:"duplicateDetection"`
	FacialRecognition struct {
		Enabled     bool    `json:"enabled"`
		ModelName   string  `json:"modelName"`
		MinScore    float64 `json:"minScore" v:"min=0.1,max=1"`
		MaxDistance float64 `json:"maxDistance" v:"min=0.1,max=2"`
		MinFaces    int64   `json:"minFaces" v:"min=1"`
	} `json:"facialRecognition"`
	OCR struct {
		Enabled             bool    `json:"enabled"`
		ModelName           string  `json:"modelName"`
		MaxResolution       int64   `json:"maxResolution" v:"min=1"`
		MinDetectionScore   float64 `json:"minDetectionScore" v:"min=0.1,max=1"`
		MinRecognitionScore float64 `json:"minRecognitionScore" v:"min=0.1,max=1"`
	} `json:"ocr"`
}

type Map struct {
	Enabled    bool   `json:"enabled" v:"configbool"`
	LightStyle string `json:"lightStyle" v:"url"`
	DarkStyle  string `json:"darkStyle" v:"url"`
}

type Toggle struct {
	Enabled bool `json:"enabled" v:"configbool"`
}

type OAuth struct {
	AutoLaunch              bool                  `json:"autoLaunch" v:"configbool"`
	AutoRegister            bool                  `json:"autoRegister" v:"configbool"`
	ButtonText              string                `json:"buttonText"`
	ClientID                string                `json:"clientId"`
	ClientSecret            string                `json:"clientSecret"`
	TokenEndpointAuthMethod AuthMethod            `json:"tokenEndpointAuthMethod"`
	Timeout                 int64                 `json:"timeout" v:"min=1"`
	AllowInsecureRequests   bool                  `json:"allowInsecureRequests" v:"configbool"`
	DefaultStorageQuota     optional.Value[int64] `json:"defaultStorageQuota" v:"required,nullable,min=0"`
	Enabled                 bool                  `json:"enabled" v:"configbool"`
	IssuerURL               string                `json:"issuerUrl" v:"refine=issuerUrl"`
	AccountManagementURL    *string               `json:"accountManagementUrl" v:"refine=accountUrl"`
	Scope                   string                `json:"scope"`
	Prompt                  string                `json:"prompt"`
	EndSessionEndpoint      string                `json:"endSessionEndpoint" v:"refine=endSessionUrl"`
	SigningAlgorithm        string                `json:"signingAlgorithm"`
	ProfileSigningAlgorithm string                `json:"profileSigningAlgorithm"`
	StorageLabelClaim       string                `json:"storageLabelClaim"`
	StorageQuotaClaim       string                `json:"storageQuotaClaim"`
	RoleClaim               string                `json:"roleClaim"`
	MobileOverrideEnabled   bool                  `json:"mobileOverrideEnabled" v:"configbool"`
	MobileRedirectURI       string                `json:"mobileRedirectUri"`
}

type StorageTemplate struct {
	Enabled                 bool   `json:"enabled" v:"configbool"`
	HashVerificationEnabled bool   `json:"hashVerificationEnabled" v:"configbool"`
	Template                string `json:"template"`
}

type Rendition struct {
	Format      ImageFormat `json:"format"`
	Quality     int64       `json:"quality" v:"min=1,max=100"`
	Size        int64       `json:"size" v:"min=1"`
	Progressive *bool       `json:"progressive" v:"configbool"`
}

type Image struct {
	Thumbnail Rendition `json:"thumbnail"`
	Preview   Rendition `json:"preview"`
	Fullsize  struct {
		Enabled     bool        `json:"enabled" v:"configbool"`
		Format      ImageFormat `json:"format"`
		Quality     int64       `json:"quality" v:"min=1,max=100"`
		Progressive *bool       `json:"progressive" v:"configbool"`
	} `json:"fullsize"`
	Colorspace      Colorspace `json:"colorspace"`
	ExtractEmbedded bool       `json:"extractEmbedded" v:"configbool"`
}

type NightlyTasks struct {
	StartTime         string `json:"startTime" v:"time"`
	DatabaseCleanup   bool   `json:"databaseCleanup" v:"configbool"`
	MissingThumbnails bool   `json:"missingThumbnails" v:"configbool"`
	ClusterNewFaces   bool   `json:"clusterNewFaces" v:"configbool"`
	GenerateMemories  bool   `json:"generateMemories" v:"configbool"`
	SyncQuotaUsage    bool   `json:"syncQuotaUsage" v:"configbool"`
}

type SMTP struct {
	Enabled   bool   `json:"enabled" v:"configbool"`
	From      string `json:"from"`
	ReplyTo   string `json:"replyTo"`
	Transport struct {
		IgnoreCert bool   `json:"ignoreCert" v:"configbool"`
		Host       string `json:"host"`
		Port       int64  `json:"port" v:"min=0,max=65535"`
		Secure     bool   `json:"secure" v:"configbool"`
		Username   string `json:"username"`
		Password   string `json:"password"`
	} `json:"transport"`
}

type Config struct {
	Backup           Backup          `json:"backup"`
	FFmpeg           FFmpeg          `json:"ffmpeg"`
	IntegrityChecks  IntegrityChecks `json:"integrityChecks"`
	Job              Jobs            `json:"job"`
	Logging          Logging         `json:"logging"`
	MachineLearning  MachineLearning `json:"machineLearning"`
	Map              Map             `json:"map"`
	ReverseGeocoding Toggle          `json:"reverseGeocoding"`
	Metadata         struct {
		Faces struct {
			Import bool `json:"import" v:"configbool"`
		} `json:"faces"`
	} `json:"metadata"`
	OAuth           OAuth           `json:"oauth"`
	PasswordLogin   Toggle          `json:"passwordLogin"`
	StorageTemplate StorageTemplate `json:"storageTemplate"`
	Image           Image           `json:"image"`
	NewVersionCheck struct {
		Enabled bool    `json:"enabled" v:"configbool"`
		Channel Channel `json:"channel"`
	} `json:"newVersionCheck"`
	NightlyTasks NightlyTasks `json:"nightlyTasks"`
	Trash        struct {
		Enabled bool  `json:"enabled" v:"configbool"`
		Days    int64 `json:"days" v:"min=0"`
	} `json:"trash"`
	Theme struct {
		CustomCSS string `json:"customCss"`
	} `json:"theme"`
	Library struct {
		Scan  Cron   `json:"scan"`
		Watch Toggle `json:"watch"`
	} `json:"library"`
	Notifications struct {
		SMTP SMTP `json:"smtp"`
	} `json:"notifications"`
	Templates struct {
		Email struct {
			WelcomeTemplate     string `json:"welcomeTemplate"`
			AlbumInviteTemplate string `json:"albumInviteTemplate"`
			AlbumUpdateTemplate string `json:"albumUpdateTemplate"`
		} `json:"email"`
	} `json:"templates"`
	Server struct {
		ExternalDomain   string `json:"externalDomain" v:"refine=externalDomain"`
		LoginPageMessage string `json:"loginPageMessage"`
		PublicUsers      bool   `json:"publicUsers" v:"configbool"`
	} `json:"server"`
	User struct {
		DeleteDelay int64 `json:"deleteDelay" v:"min=1"`
	} `json:"user"`
}
