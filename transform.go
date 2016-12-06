package imgwizard

import (
	"net/http"

	"github.com/shifr/vips"
)

func Transform(img_buff []byte, ctx *Context) []byte {
	debug("Detecting image type...")
	iType := http.DetectContentType(img_buff)

	if !stringExists(iType, ResizableImageTypes) {
		warning("Wizard resize doesn't support image type, returning original image")
		return img_buff
	}

	if ctx.IsOriginal {
		debug("Returning original image as requested...")
		return img_buff
	}

	buf, err := vips.Resize(img_buff, ctx.Options)
	if err != nil {
		warning("Can't resize img, reason - %s", err)
		return img_buff
	}

	return buf
}
