package useragent

import "testing"

func TestParseMatchesUAParserJS(t *testing.T) {
	tests := []struct {
		ua         string
		deviceType string
		deviceOS   string
	}{
		{"immich-android/3.3.3", "", "android"},
		{"immich-ios/3.3.3", "", ""},
		{"immich-unknown/3.3.3", "", ""},
		{"Immich_Android_1.106.0", "", "Android"},
		{"Immich_iOS_1.106.0", "", ""},
		{"curl/8.11.1", "", ""},
		{"Dart/3.9 (dart:io)", "", ""},
		{"okhttp/4.12.0", "", ""},
		{"", "", ""},
		{"Mozilla/5.0 (X11; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0", "Firefox", "Linux"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36", "Chrome", "Windows"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Edg/141.0.0.0", "Edge", "Windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.6 Safari/605.1.15", "Safari", "macOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.6 Mobile/15E148 Safari/604.1", "Mobile Safari", "iOS"},
		{"Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36", "Mobile Chrome", "Android"},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 OPR/120.0.0.0", "Opera", "Linux"},
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chromium/141.0.0.0 Chrome/141.0.0.0 Safari/537.36", "Chromium", "Linux"},
		{"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0", "Firefox", "Ubuntu"},
		{"Mozilla/5.0 (X11; Fedora; Linux x86_64; rv:140.0) Gecko/20100101 Firefox/140.0", "Firefox", "Fedora"},
		{"Mozilla/5.0 (iPad; CPU OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.6 Mobile/15E148 Safari/604.1", "Mobile Safari", "iOS"},
		{"Mozilla/5.0 (Linux; Android 15; SM-S928B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/28.0 Chrome/130.0.0.0 Mobile Safari/537.36", "Samsung Internet", "Android"},
		{"Mozilla/5.0 (Android 15; Mobile; rv:140.0) Gecko/140.0 Firefox/140.0", "Mobile Firefox", "Android"},
		{"Mozilla/5.0 (X11; CrOS x86_64 16328.65.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36", "Chrome", "Chrome OS"},
		{"Mozilla/5.0 (X11; FreeBSD amd64; rv:140.0) Gecko/20100101 Firefox/140.0", "Firefox", "FreeBSD"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:140.0) Gecko/20100101 Firefox/140.0", "Firefox", "Windows"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:140.0) Gecko/20100101 Firefox/140.0", "Firefox", "macOS"},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36", "Chrome", "macOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/141.0.0.0 Mobile/15E148 Safari/604.1", "Mobile Chrome", "iOS"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 18_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/140.0 Mobile/15E148 Safari/605.1.15", "Mobile Firefox", "iOS"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Vivaldi/7.5", "Vivaldi", "Windows"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36 Brave/141", "Brave", "Windows"},
		{"python-requests/2.32.3", "", ""},
		{"Go-http-client/1.1", "", ""},
		{"PostmanRuntime/7.43.0", "", ""},
		{"Wget/1.25.0", "", ""},
	}

	for _, tt := range tests {
		got := Parse(tt.ua, "", "")
		if got.DeviceType != tt.deviceType || got.DeviceOS != tt.deviceOS {
			t.Errorf("Parse(%q) = %q, %q, want %q, %q", tt.ua, got.DeviceType, got.DeviceOS, tt.deviceType, tt.deviceOS)
		}
	}
}

func TestParseFallsBackToTheAppsHeadersAndReadsItsVersion(t *testing.T) {
	got := Parse("immich-ios/3.3.3", "iPhone17,1", "iOS")
	if got.DeviceType != "iPhone17,1" || got.DeviceOS != "iOS" || !got.AppVersion.Valid || got.AppVersion.V != "3.3.3" {
		t.Errorf("Parse of the iOS app = %+v", got)
	}

	got = Parse("immich-android/3.3.3", "Pixel 9", "Android")
	if got.DeviceType != "Pixel 9" || got.DeviceOS != "android" || got.AppVersion.V != "3.3.3" {
		t.Errorf("Parse of the Android app = %+v", got)
	}

	if got = Parse("Immich_Android_1.106.0", "", ""); got.AppVersion.V != "1.106.0" {
		t.Errorf("the legacy agent gave the version %q", got.AppVersion.V)
	}

	if got = Parse("curl/8.11.1", "", ""); got.AppVersion.Valid || got.DeviceType != "" || got.DeviceOS != "" {
		t.Errorf("Parse of curl = %+v", got)
	}
}
