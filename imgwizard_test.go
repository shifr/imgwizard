package main

import "testing"

func TestCachePath(t *testing.T) {
	tests := []struct {
		CacheDir  string
		Storage   string
		Path      string
		Width     int
		Height    int
		CachePath string
	}{
		{
			"/tmp/imgwizard",
			"rem",
			"media.somesite.ua/uploads/product_image/2015/02/242/c6cc5dd1-6f25-4642-734d-bbf5bef5dffa_8b91c9c17027e331f991ddc7ea3b2dd9_orig.jpg",
			320,
			240,
			"/tmp/imgwizard/media.somesite.ua/uploads/product_image/2015/02/242/c6cc5dd1-6f25-4642-734d-bbf5bef5dffa_8b91c9c17027e331f991ddc7ea3b2dd9_orig_320x240.jpg",
		},
		{
			"/your_cache_path",
			"rem",
			"media.somesite.ua/uploads/product_image/2015/02/242/c6cc5dd1-6f25-4642-734d-bbf5bef5dffa_8b91c9c17027e331f991ddc7ea3b2dd9_orig.jpg",
			320,
			240,
			"/your_cache_path/media.somesite.ua/uploads/product_image/2015/02/242/c6cc5dd1-6f25-4642-734d-bbf5bef5dffa_8b91c9c17027e331f991ddc7ea3b2dd9_orig_320x240.jpg",
		},
		{
			"/tmp/imgwizard",
			"loc",
			"media.somesite.ua/uploads/product_image/2015/02/242/c6cc5dd1-6f25-4642-734d-bbf5bef5dffa_8b91c9c17027e331f991ddc7ea3b2dd9_orig.jpg",
			320,
			240,
			"/tmp/imgwizard/media.somesite.ua/uploads/product_image/2015/02/242/c6cc5dd1-6f25-4642-734d-bbf5bef5dffa_8b91c9c17027e331f991ddc7ea3b2dd9_orig_320x240.jpg",
		},
		{
			"/tmp/imgwizard",
			"loc",
			"media_dir/image_orig.jpg",
			320,
			240,
			"/tmp/imgwizard/media_dir/image_orig_320x240.jpg",
		},
		{
			"/tmp/imgwizard",
			"loc",
			"m/e/d/i/a_dir/image_orig.jpg",
			320,
			240,
			"/tmp/imgwizard/m/e/d/i/a_dir/image_orig_320x240.jpg",
		},
		{
			"/tmp/imgwizard",
			"rem",
			"m/e/d/i/a_dir/image_orig.jpg",
			0,
			240,
			"/tmp/imgwizard/m/e/d/i/a_dir/image_orig_0x240.jpg",
		},
		{
			"/tmp/imgwizard",
			"loc",
			"m/e/d/i/a_dir/image_orig.jpg",
			0,
			0,
			"/tmp/imgwizard/m/e/d/i/a_dir/image_orig_0x0.jpg",
		},
		{
			"/tmp/imgwizard",
			"loc",
			"m/e/d/i/a_dir/image_orig.jpg?crop=left,top&q=10",
			0,
			0,
			"/tmp/imgwizard/m/e/d/i/a_dir/image_orig_0x0.jpg?crop=left,top&q=10",
		},
	}

	for i, test := range tests {
		context := Context{}

		CacheDir = test.CacheDir
		context.Storage = test.Storage
		context.Path = test.Path
		context.Options.Width = test.Width
		context.Options.Height = test.Height

		context.makeCachePath()

		CachePath := context.CachePath

		if test.CachePath != CachePath {
			t.Errorf("%d. makeCachePath returned %v, needed %v", i, CachePath, test.CachePath)
		}
	}
}
