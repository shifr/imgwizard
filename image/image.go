package imgwizard

import (
	"github.com/shifr/imgwizard"
	"github.com/shifr/vips"
)

func Transform(img_buff []byte, ctx *imgwizard.Context, iType string) []byte {
	buf, err := vips.Resize(img, ctx.Options)
	if err != nil {
		warning("Can't resize img, reason - %s", err)

		err = Cache.Set(ctx.CachePath, img)
		if err != nil {
			warning("Can't set cache, reason - %s", err)
		}
		return img
	}
}
