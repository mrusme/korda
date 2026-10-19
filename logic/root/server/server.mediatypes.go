package server

type MediaTypes struct {
	Video   []string
	Image   []string
	Sidecar []string
}

var (
	webImages = []string{".avif", ".bmp", ".gif", ".jfif", ".jpeg", ".jpg", ".png", ".webp"}
	rawImages = []string{
		".3fr", ".ari", ".arw", ".cap", ".cin", ".cr2", ".cr3", ".crw", ".dcr", ".dng",
		".erf", ".fff", ".iiq", ".k25", ".kdc", ".mrw", ".nef", ".nrw", ".orf", ".ori",
		".pef", ".psd", ".raf", ".raw", ".rw2", ".rwl", ".sr2", ".srf", ".srw", ".x3f",
	}
	otherImages = []string{".heic", ".heif", ".hif", ".insp", ".jp2", ".jpe", ".jxl", ".mpo", ".svg", ".tif", ".tiff"}
	videos      = []string{
		".3gp", ".3gpp", ".avi", ".flv", ".insv", ".m2t", ".m2ts", ".m4v", ".mkv", ".mov",
		".mp4", ".mpe", ".mpeg", ".mpg", ".mts", ".mxf", ".ts", ".vob", ".webm", ".wmv",
	}
	sidecars = []string{".xmp"}
)

func GetMediaTypes() MediaTypes {
	images := make([]string, 0, len(webImages)+len(rawImages)+len(otherImages))
	images = append(images, webImages...)
	images = append(images, rawImages...)
	images = append(images, otherImages...)

	return MediaTypes{
		Video:   append([]string(nil), videos...),
		Image:   images,
		Sidecar: append([]string(nil), sidecars...),
	}
}
