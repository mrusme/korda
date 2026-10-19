package server

import (
	"time"

	"xn--gckvb8fzb.com/glides/runtime"
)

type About struct {
	Version       string
	VersionURL    string
	Licensed      bool
	Repository    string
	RepositoryURL string
	SourceRef     string
	SourceCommit  string
	SourceURL     string
}

type APKLinks struct {
	Arm64v8a   string
	Armeabiv7a string
	Universal  string
	X86_64     string
}

func now() time.Time {
	return time.Now().UTC()
}

func GetAbout(rt *runtime.Runtime) About {
	version, commit, _, _ := rt.GetBuild()
	if version == "" {
		version = "dev"
	}

	about := About{
		Version:       version,
		VersionURL:    RepositoryURL + "/releases",
		Licensed:      false,
		Repository:    "mrus/korda",
		RepositoryURL: RepositoryURL,
		SourceRef:     version,
	}
	if commit != "" {
		about.SourceCommit = commit
		about.SourceURL = RepositoryURL + "/commit/" + commit
	}

	return about
}

func GetAPKLinks() APKLinks {
	base := "https://github.com/immich-app/immich/releases/download/v" + ImmichVersion + "/"

	return APKLinks{
		Arm64v8a:   base + "app-arm64-v8a-release.apk",
		Armeabiv7a: base + "app-armeabi-v7a-release.apk",
		Universal:  base + "app-release.apk",
		X86_64:     base + "app-x86_64-release.apk",
	}
}
