package compress

import (
	"regexp"
	"strings"
)

var (
	extractType      = regexp.MustCompile(`^\s*([^;\s]*)(?:;|\s|$)`)
	compressibleType = regexp.MustCompile(`(?i)^text/|\+(?:json|text|xml)$`)
	compressibleList = map[string]bool{
		"application/dart":                  true,
		"application/ecmascript":            true,
		"application/javascript":            true,
		"application/json":                  true,
		"application/octet-stream":          true,
		"application/postscript":            true,
		"application/raml+yaml":             true,
		"application/rtf":                   true,
		"application/tar":                   true,
		"application/toml":                  true,
		"application/vnd.dart":              true,
		"application/vnd.ms-fontobject":     true,
		"application/vnd.ms-opentype":       true,
		"application/wasm":                  true,
		"application/x-httpd-php":           true,
		"application/x-javascript":          true,
		"application/x-ns-proxy-autoconfig": true,
		"application/x-sh":                  true,
		"application/x-tar":                 true,
		"application/x-virtualbox-hdd":      true,
		"application/x-virtualbox-ova":      true,
		"application/x-virtualbox-ovf":      true,
		"application/x-virtualbox-vbox":     true,
		"application/x-virtualbox-vdi":      true,
		"application/x-virtualbox-vhd":      true,
		"application/x-virtualbox-vmdk":     true,
		"application/x-www-form-urlencoded": true,
		"application/xml":                   true,
		"application/xml-dtd":               true,
		"font/otf":                          true,
		"font/ttf":                          true,
		"image/bmp":                         true,
		"image/vnd.adobe.photoshop":         true,
		"image/vnd.microsoft.icon":          true,
		"image/vnd.ms-dds":                  true,
		"image/x-icon":                      true,
		"image/x-ms-bmp":                    true,
		"message/rfc822":                    true,
		"model/gltf-binary":                 true,
		"x-shader/x-fragment":               true,
		"x-shader/x-vertex":                 true,
	}
)

func Compressible(contentType string) bool {
	m := extractType.FindStringSubmatch(contentType)
	if m == nil {
		return false
	}
	mime := strings.ToLower(m[1])

	return compressibleList[mime] || compressibleType.MatchString(mime)
}
