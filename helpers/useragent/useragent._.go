package useragent

import (
	"database/sql"
	"regexp"
	"strings"
)

type Details struct {
	DeviceType string
	DeviceOS   string
	AppVersion sql.Null[string]
}

var appVersions = []*regexp.Regexp{
	regexp.MustCompile(`^immich-(?:android|ios|unknown)/(.+)$`),
	regexp.MustCompile(`^Immich_(?:Android|iOS|Unknown)_(.+)$`),
}

var (
	androidOS = regexp.MustCompile(`(?i)android`)
	distros   = regexp.MustCompile(`(?i)\b(ubuntu|fedora|debian|mint|centos|gentoo|arch linux|opensuse|suse|red hat|freebsd|openbsd|netbsd)\b`)
)

func Parse(ua string, deviceModel string, deviceType string) Details {
	d := Details{
		DeviceType: firstOf(browser(ua), device(ua), deviceModel),
		DeviceOS:   firstOf(system(ua), deviceType),
	}

	for _, re := range appVersions {
		if m := re.FindStringSubmatch(ua); m != nil {
			d.AppVersion = sql.Null[string]{V: m[1], Valid: true}
			break
		}
	}

	return d
}

func firstOf(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

func browser(ua string) string {
	has := func(s string) bool { return strings.Contains(ua, s) }

	switch {
	case has("Edg/"), has("EdgA/"), has("EdgiOS/"):
		return "Edge"
	case has("OPR/"):
		return "Opera"
	case has("SamsungBrowser/"):
		return "Samsung Internet"
	case has("Vivaldi/"):
		return "Vivaldi"
	case has("Brave/"):
		return "Brave"
	case has("CriOS/"):
		return "Mobile Chrome"
	case has("FxiOS/"):
		return "Mobile Firefox"
	case has("Chromium/"):
		return "Chromium"
	case has("Firefox/"):
		if has("Mobile") || has("Tablet") {
			return "Mobile Firefox"
		}
		return "Firefox"
	case has("Chrome/"):
		if has("Mobile Safari/") {
			return "Mobile Chrome"
		}
		return "Chrome"
	case has("Version/") && has("Safari/"):
		if has("Mobile/") {
			return "Mobile Safari"
		}
		return "Safari"
	}

	return ""
}

func device(ua string) string {
	has := func(s string) bool { return strings.Contains(ua, s) }

	switch {
	case has("iPad"):
		return "tablet"
	case has("iPhone"), has("iPod"):
		return "mobile"
	case has("Android") && strings.Contains(ua, "Mozilla/"):
		if has("Mobile") {
			return "mobile"
		}
		return "tablet"
	}

	return ""
}

func system(ua string) string {
	has := func(s string) bool { return strings.Contains(ua, s) }

	if m := androidOS.FindString(ua); m != "" {
		return m
	}

	switch {
	case has("CrOS"):
		return "Chrome OS"
	case has("Windows NT"), has("Windows Phone"):
		return "Windows"
	case has("iPhone OS"), has("iPad"), has("iPod"):
		return "iOS"
	case has("Mac OS X"), has("Macintosh"):
		return "macOS"
	}

	if m := distros.FindString(ua); m != "" {
		return m
	}
	if has("Linux") {
		return "Linux"
	}

	return ""
}
