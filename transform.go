package imgwizard

import (
	"bytes"
	"image"
	"image/png"
	"net/http"

	"github.com/foobaz/lossypng/lossypng"
	"github.com/shifr/vips"
)

func Transform(img_buff *[]byte, ctx *Context) {
	var err error
	buf := new(bytes.Buffer)

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
		decoded, _, err := image.Decode(bytes.NewReader(*img_buff))
		if err != nil {
			warning("Can't decode PNG image, reason - %s", err)
		}

		out := lossypng.Compress(decoded, lossypng.NoConversion, 100-ctx.Options.Quality)
		err = png.Encode(buf, out)
		if err != nil {
			warning("Can't encode PNG image, reason - %s", err)
		}

		*img_buff = buf.Bytes()
	}

}
