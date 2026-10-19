package mimetypes

import (
	"path/filepath"
	"slices"
	"strings"
)

const OctetStream = "application/octet-stream"

type entry struct {
	ext   string
	types []string
}

var raw = []entry{
	{".3fr", []string{"image/3fr", "image/x-hasselblad-3fr"}},
	{".ari", []string{"image/ari", "image/x-arriflex-ari"}},
	{".arw", []string{"image/arw", "image/x-sony-arw"}},
	{".cap", []string{"image/cap", "image/x-phaseone-cap"}},
	{".cin", []string{"image/cin", "image/x-phantom-cin"}},
	{".cr2", []string{"image/cr2", "image/x-canon-cr2"}},
	{".cr3", []string{"image/cr3", "image/x-canon-cr3"}},
	{".crw", []string{"image/crw", "image/x-canon-crw"}},
	{".dcr", []string{"image/dcr", "image/x-kodak-dcr"}},
	{".dng", []string{"image/dng", "image/x-adobe-dng"}},
	{".erf", []string{"image/erf", "image/x-epson-erf"}},
	{".fff", []string{"image/fff", "image/x-hasselblad-fff"}},
	{".iiq", []string{"image/iiq", "image/x-phaseone-iiq"}},
	{".k25", []string{"image/k25", "image/x-kodak-k25"}},
	{".kdc", []string{"image/kdc", "image/x-kodak-kdc"}},
	{".mrw", []string{"image/mrw", "image/x-minolta-mrw"}},
	{".nef", []string{"image/nef", "image/x-nikon-nef"}},
	{".nrw", []string{"image/nrw", "image/x-nikon-nrw"}},
	{".orf", []string{"image/orf", "image/x-olympus-orf"}},
	{".ori", []string{"image/ori", "image/x-olympus-ori"}},
	{".pef", []string{"image/pef", "image/x-pentax-pef"}},
	{".psd", []string{"image/psd", "image/vnd.adobe.photoshop"}},
	{".raf", []string{"image/raf", "image/x-fuji-raf"}},
	{".raw", []string{"image/raw", "image/x-panasonic-raw"}},
	{".rw2", []string{"image/rw2", "image/x-panasonic-rw2"}},
	{".rwl", []string{"image/rwl", "image/x-leica-rwl"}},
	{".sr2", []string{"image/sr2", "image/x-sony-sr2"}},
	{".srf", []string{"image/srf", "image/x-sony-srf"}},
	{".srw", []string{"image/srw", "image/x-samsung-srw"}},
	{".x3f", []string{"image/x3f", "image/x-sigma-x3f"}},
}

var webImage = []entry{
	{".avif", []string{"image/avif"}},
	{".bmp", []string{"image/bmp"}},
	{".gif", []string{"image/gif"}},
	{".jfif", []string{"image/jpeg"}},
	{".jpeg", []string{"image/jpeg"}},
	{".jpg", []string{"image/jpeg"}},
	{".png", []string{"image/png", "image/apng"}},
	{".webp", []string{"image/webp"}},
}

var otherImage = []entry{
	{".heic", []string{"image/heic"}},
	{".heif", []string{"image/heif"}},
	{".hif", []string{"image/hif"}},
	{".insp", []string{"image/jpeg"}},
	{".jp2", []string{"image/jp2"}},
	{".jpe", []string{"image/jpeg"}},
	{".jxl", []string{"image/jxl"}},
	{".mpo", []string{"image/jpeg"}},
	{".svg", []string{"image/svg"}},
	{".tif", []string{"image/tiff"}},
	{".tiff", []string{"image/tiff"}},
}

var video = []entry{
	{".3gp", []string{"video/3gpp"}},
	{".3gpp", []string{"video/3gpp"}},
	{".avi", []string{"video/avi", "video/msvideo", "video/vnd.avi", "video/x-msvideo"}},
	{".flv", []string{"video/x-flv"}},
	{".insv", []string{"video/mp4"}},
	{".m2t", []string{"video/mp2t"}},
	{".m2ts", []string{"video/mp2t"}},
	{".m4v", []string{"video/x-m4v"}},
	{".mkv", []string{"video/x-matroska"}},
	{".mov", []string{"video/quicktime"}},
	{".mp4", []string{"video/mp4"}},
	{".mpe", []string{"video/mpeg"}},
	{".mpeg", []string{"video/mpeg"}},
	{".mpg", []string{"video/mpeg"}},
	{".mts", []string{"video/mp2t"}},
	{".mxf", []string{"application/mxf"}},
	{".ts", []string{"video/mp2t"}},
	{".vob", []string{"video/mpeg"}},
	{".webm", []string{"video/webm"}},
	{".wmv", []string{"video/x-ms-wmv"}},
}

var sidecar = []entry{
	{".xmp", []string{"application/xml", "text/xml"}},
}

var (
	heif     = []string{".avif", ".heic", ".heif", ".hif"}
	animated = []string{".avif", ".gif", ".heic", ".heif", ".jxl", ".png", ".webp"}
)

func Extension(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

func Base(filename string) string {
	name := filepath.Base(filename)
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func find(lists [][]entry, filename string) *entry {
	ext := Extension(filename)
	for _, list := range lists {
		for i := range list {
			if list[i].ext == ext {
				return &list[i]
			}
		}
	}

	return nil
}

func Lookup(filename string) string {
	if e := find([][]entry{webImage, raw, otherImage, video, sidecar}, filename); e != nil {
		return e.types[0]
	}

	return OctetStream
}

func ToExtension(mimeType string) string {
	if mimeType == "image/jpeg" {
		return ".jpg"
	}

	for _, list := range [][]entry{webImage, raw, otherImage, video, sidecar} {
		for _, e := range list {
			if slices.Contains(e.types, mimeType) {
				return e.ext
			}
		}
	}

	return ""
}

func IsImage(filename string) bool {
	return find([][]entry{webImage, raw, otherImage}, filename) != nil
}

func IsVideo(filename string) bool {
	return find([][]entry{video}, filename) != nil
}

func IsAsset(filename string) bool {
	return IsImage(filename) || IsVideo(filename)
}

func IsRaw(filename string) bool {
	return find([][]entry{raw}, filename) != nil
}

func IsWebSupportedImage(filename string) bool {
	return find([][]entry{webImage}, filename) != nil
}

func IsSidecar(filename string) bool {
	return find([][]entry{sidecar}, filename) != nil
}

func IsHEIF(filename string) bool {
	return slices.Contains(heif, Extension(filename))
}

func IsPossiblyAnimated(filename string) bool {
	return slices.Contains(animated, Extension(filename))
}

func extensions(lists ...[]entry) []string {
	var out []string
	for _, list := range lists {
		for _, e := range list {
			out = append(out, e.ext)
		}
	}

	return out
}

func ImageExtensions() []string {
	return extensions(webImage, raw, otherImage)
}

func VideoExtensions() []string {
	return extensions(video)
}

func SidecarExtensions() []string {
	return extensions(sidecar)
}
