package imgwizard

import (
	"net/http"

	"github.com/shifr/goquant"
	"github.com/shifr/vips"
)

func Transform(img_buff *[]byte, ctx *Context) {
	var err error

	debug("Detecting image type...")
	iType := http.DetectContentType(*img_buff)

	if !stringExists(iType, ResizableImageTypes) {
		warning("Wizard resize doesn't support image type, returning original image")
		return
	}

	*img_buff, err = vips.Resize(*img_buff, ctx.Options)
	if err != nil {
		warning("Can't resize img, reason - %s", err)
		return
	}

	if iType == PNG && !ctx.Options.Webp {
		goquant.Quantize(img_buff)
		debug("NEW IMAGE SIZE: %d", len(*img_buff))
	}
}
