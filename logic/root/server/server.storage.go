package server

import (
	gh "tty.fail/mrus/korda/helpers"
	"tty.fail/mrus/korda/services/mediaroot"
	"xn--gckvb8fzb.com/glides/runtime"
)

func GetStorage(rt *runtime.Runtime) (mediaroot.Usage, error) {
	return gh.MediaRoot(rt).Usage()
}
