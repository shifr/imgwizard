package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/shifr/imgwizard"
)

const VERSION = 1.5

func init() {
	log.SetOutput(os.Stdout)

	flag.BoolVar(&imgwizard.Version, "v", false, "Check imgwizard version")
	flag.StringVar(&imgwizard.ListenAddr, "l", "127.0.0.1:8070", "Address to listen on")
	flag.StringVar(&imgwizard.AllowedMedia, "m", "", "comma separated list of allowed media server hosts")
	flag.StringVar(&imgwizard.AllowedSizes, "s", "", "comma separated list of allowed sizes")
	flag.StringVar(&imgwizard.CacheDir, "c", "/tmp/imgwizard", "directory for cached files")
	flag.StringVar(&imgwizard.S3BucketName, "s3-b", "", "AWS S3 cache bucket name")
	flag.StringVar(&imgwizard.AzureContainerName, "az", "", "Microsoft Azure Storage container name")
	flag.StringVar(&imgwizard.Default404, "thumb", "", "path to default image if original not found")
	flag.StringVar(&imgwizard.DirsToSearch, "d", "", "comma separated list of directories to search requested file")
	flag.StringVar(&imgwizard.Mark, "mark", "images", "Mark for nginx")
	flag.StringVar(&imgwizard.NoCacheKey, "no-cache-key", "", "Secret key that must be equal X-No-Cache value from request header")
	flag.StringVar(&imgwizard.Nodes, "nodes", "", "Other imgwizard nodes to ask before process image")
	flag.IntVar(&imgwizard.Quality, "q", 0, "image quality after resize")
}

func main() {
	flag.Parse()

	if imgwizard.Version {
		log.Println("Version:", VERSION)
		return
	}

	imgwizard.GlobalSettings.Load()

	r := new(imgwizard.RegexpHandler)
	r.HandleFunc(imgwizard.GlobalSettings.UrlExp, imgwizard.FetchImage)

	log.Printf("ImgWizard started on http://%s", imgwizard.ListenAddr)
	http.ListenAndServe(imgwizard.ListenAddr, r)
}
