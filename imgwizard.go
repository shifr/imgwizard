package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/shifr/imgwizard/cache"

	"github.com/gorilla/mux"
	"github.com/shifr/vips"
)

type Context struct {
	Path      string
	Format    string
	CachePath string
	Storage   string
	Width     int
	Height    int
}

type Settings struct {
	ListenAddr    string
	CacheDir      string
	Scheme        string
	Local404Thumb string
	AllowedSizes  []string
	AllowedMedia  []string
	UrlTemplate   string

	Context Context
	Options vips.Options
}

const (
	DEFAULT_CACHE_DIR = "/tmp/imgwizard"
	WEBP_HEADER       = "image/webp"
)

var (
	settings         Settings
	supportedFormats = []string{"jpg", "jpeg", "png"}
	listenAddr       = flag.String("l", "127.0.0.1:8070", "Address to listen on")
	allowedMedia     = flag.String("m", "", "comma separated list of allowed media")
	allowedSizes     = flag.String("s", "", "comma separated list of allowed sizes")
	cacheDir         = flag.String("c", "", "directory for cached files")
	mark             = flag.String("mark", "images", "Mark for nginx")
	quality          = flag.Int("q", 0, "image quality after resize")
)

// loadSettings loads settings from settings.json
// and from command-line
func (s *Settings) loadSettings() {

	s.CacheDir = DEFAULT_CACHE_DIR
	s.Scheme = "http"
	s.Local404Thumb = "/tmp/404.jpg"
	s.AllowedSizes = nil
	s.AllowedMedia = nil

	//defaults for vips
	s.Options.Crop = true
	s.Options.Enlarge = true
	s.Options.Quality = 80
	s.Options.Extend = vips.EXTEND_WHITE
	s.Options.Interpolator = vips.BILINEAR
	s.Options.Gravity = vips.CENTRE

	var sizes = "[0-9]*x[0-9]*"
	var medias = ""
	var proxyMark = *mark

	s.ListenAddr = *listenAddr

	if *allowedMedia != "" {
		s.AllowedMedia = strings.Split(*allowedMedia, ",")
	}

	if *allowedSizes != "" {
		s.AllowedSizes = strings.Split(*allowedSizes, ",")
	}

	if *cacheDir != "" {
		s.CacheDir = *cacheDir
	}

	if *quality != 0 {
		s.Options.Quality = *quality
	}

	if len(s.AllowedSizes) > 0 {
		sizes = strings.Join(s.AllowedSizes, "|")
	}

	if len(s.AllowedMedia) > 0 {
		medias = strings.Join(s.AllowedMedia, "|")
	}

	s.UrlTemplate = fmt.Sprintf(
		"/{mark:%s}/{storage:loc|rem}/{size:%s}/{path:%s.+}", proxyMark, sizes, medias)
}

// makeCachePath generates cache path from resized image
func (s *Settings) makeCachePath() {
	var subPath string
	var cacheImageName string

	pathParts := strings.Split(s.Context.Path, "/")
	lastIndex := len(pathParts) - 1
	imageData := strings.Split(pathParts[lastIndex], ".")
	imageName, imageFormat := imageData[0], strings.ToLower(imageData[1])

	if s.Options.Webp {
		cacheImageName = fmt.Sprintf(
			"%s_%dx%d_webp_.%s", imageName, s.Options.Width, s.Options.Height, imageFormat)
	} else {
		cacheImageName = fmt.Sprintf(
			"%s_%dx%d.%s", imageName, s.Options.Width, s.Options.Height, imageFormat)
	}

	switch s.Context.Storage {
	case "loc":
		subPath = strings.Join(pathParts[:lastIndex], "/")
	case "rem":
		subPath = strings.Join(pathParts[1:lastIndex], "/")
	}
	s.Context.Format = imageFormat
	s.Context.CachePath = fmt.Sprintf(
		"%s/%s/%s", s.CacheDir, subPath, cacheImageName)
}

// getLocalImage fetches original image from file system
func getLocalImage(s *Settings) ([]byte, error) {
	var image []byte

	file, err := os.Open(path.Join("/", s.Context.Path))
	if err != nil {
		file, err = os.Open(s.Local404Thumb)
		if err != nil {
			return image, err
		}
	}

	info, _ := file.Stat()
	image = make([]byte, info.Size())

	_, err = file.Read(image)
	if err != nil {
		return image, err
	}

	return image, nil
}

// getRemoteImage fetches original image by http url
func getRemoteImage(url string) ([]byte, error) {
	var image []byte

	resp, err := http.Get(url)
	if err != nil {
		return image, err
	}
	defer resp.Body.Close()

	image, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return image, err
	}

	return image, nil
}

// getOrCreateImage check cache path for requested image
// if image doesn't exist - creates it
func getOrCreateImage() []byte {
	sett := settings
	sett.makeCachePath()

	var c *cache.Cache
	var image []byte
	var err error

	if image, err = c.Get(sett.Context.CachePath); err == nil {
		return image
	}

	switch sett.Context.Storage {
	case "loc":
		image, err = getLocalImage(&sett)
		if err != nil {
			log.Println("Can't get orig local file, reason - ", err)
		}

	case "rem":
		imgUrl := fmt.Sprintf("%s://%s", sett.Scheme, sett.Context.Path)
		image, err = getRemoteImage(imgUrl)
		if err != nil {
			log.Println("Can't get orig remote file, reason - ", err)
		}
	}

	if !stringIsExists(sett.Context.Format, supportedFormats) {
		err = c.Set(sett.Context.CachePath, image)
		if err != nil {
			log.Println("Can't set cache, reason - ", err)
		}
		return image
	}

	buf, err := vips.Resize(image, sett.Options)
	if err != nil {
		log.Println("Can't resize image, reason - ", err)
	}

	err = c.Set(sett.Context.CachePath, buf)
	if err != nil {
		log.Println("Can't set cache, reason - ", err)
	}

	return buf
}

func stringIsExists(str string, list []string) bool {
	for _, el := range list {
		if el == str {
			return true
		}
	}
	return false
}

func fetchImage(rw http.ResponseWriter, req *http.Request) {
	acceptedTypes := strings.Split(req.Header["Accept"][0], ",")

	settings.Options.Webp = stringIsExists(WEBP_HEADER, acceptedTypes)
	params := mux.Vars(req)
	sizes := strings.Split(params["size"], "x")

	settings.Context.Storage = params["storage"]
	settings.Context.Path = params["path"]
	settings.Options.Width, _ = strconv.Atoi(sizes[0])
	settings.Options.Height, _ = strconv.Atoi(sizes[1])

	resultImage := getOrCreateImage()

	rw.Write(resultImage)
}

func main() {
	flag.Parse()
	settings.loadSettings()

	r := mux.NewRouter()
	r.HandleFunc(settings.UrlTemplate, fetchImage).Methods("GET")

	log.Printf("ImgWizard started on http://%s", settings.ListenAddr)
	http.ListenAndServe(settings.ListenAddr, r)
}
