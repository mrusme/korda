package server

import "tty.fail/mrus/korda/helpers/mimetypes"

type MediaTypes struct {
	Video   []string
	Image   []string
	Sidecar []string
}

func GetMediaTypes() MediaTypes {
	return MediaTypes{
		Video:   mimetypes.VideoExtensions(),
		Image:   mimetypes.ImageExtensions(),
		Sidecar: mimetypes.SidecarExtensions(),
	}
}
