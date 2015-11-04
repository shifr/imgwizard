package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/shifr/imgwizard/cache"
	"github.com/shifr/vips"
)

type Route struct {
	pattern *regexp.Regexp
	handler http.Handler
}

type RegexpHandler struct {
	routes []*Route
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &Route{pattern, http.HandlerFunc(handler)})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

type Context struct {
	NoCache    bool
	OnlyCache  bool
	Path       string
	RequestURI string
	Format     string
	CachePath  string
	Storage    string
	Query      string
	Width      int
	Height     int
}

type Settings struct {
	ListenAddr   string
	CacheDir     string
	Scheme       string
	NoCacheKey   string
	AllowedSizes []string
	AllowedMedia []string
	Directories  []string
	Nodes        []string
	UrlExp       *regexp.Regexp

	Context Context
	Options vips.Options
}

const (
	VERSION           = 1.2
	DEFAULT_POOL_SIZE = 100000
	WEBP_HEADER       = "image/webp"
	ONLY_CACHE_HEADER = "X-Cache-Only"
	NO_CACHE_HEADER   = "X-No-Cache"
)

var (
	DEBUG            = false
	WARNING          = false
	DEFAULT_QUALITY  = 80
	ChanPool         chan int
	settings         Settings
	supportedFormats = []string{"jpg", "jpeg", "png"}
	Crop             = map[string]vips.Gravity{
		"top":    vips.NORTH,
		"right":  vips.EAST,
		"bottom": vips.SOUTH,
		"left":   vips.WEST,
	}
	listenAddr   = flag.String("l", "127.0.0.1:8070", "Address to listen on")
	allowedMedia = flag.String("m", "", "comma separated list of allowed media server hosts")
	allowedSizes = flag.String("s", "", "comma separated list of allowed sizes")
	cacheDir     = flag.String("c", "/tmp/imgwizard", "directory for cached files")
	dirsToSearch = flag.String("d", "", "comma separated list of directories to search requested file")
	mark         = flag.String("mark", "images", "Mark for nginx")
	noCacheKey   = flag.String("no-cache-key", "", "Secret key that must be equal X-No-Cache value from request header")
	quality      = flag.Int("q", 0, "image quality after resize")
	nodes        = flag.String("nodes", "", "Other imgwizard nodes to ask before process image")
)

// loadSettings loads settings from settings.json
// and from command-line
func (s *Settings) loadSettings() {

	s.Scheme = "http"
	s.AllowedSizes = nil
	s.AllowedMedia = nil

	//defaults for vips
	s.Options.Crop = true
	s.Options.Enlarge = true
	s.Options.Extend = vips.EXTEND_WHITE
	s.Options.Interpolator = vips.BILINEAR

	var sizes = "[0-9]*x[0-9]*"
	var medias = ""
	var proxyMark = *mark

	s.ListenAddr = *listenAddr
	s.CacheDir = *cacheDir

	if *allowedMedia != "" {
		s.AllowedMedia = strings.Split(*allowedMedia, ",")
	}

	if *allowedSizes != "" {
		s.AllowedSizes = strings.Split(*allowedSizes, ",")
	}

	if *dirsToSearch != "" {
		s.Directories = strings.Split(*dirsToSearch, ",")
	}

	if *noCacheKey != "" {
		s.NoCacheKey = *noCacheKey
	}

	if *nodes != "" {
		s.Nodes = strings.Split(*nodes, ",")
	}

	if *quality != 0 {
		DEFAULT_QUALITY = *quality
	}
	s.Options.Quality = DEFAULT_QUALITY

	if len(s.AllowedSizes) > 0 {
		sizes = strings.Join(s.AllowedSizes, "|")
	}

	if len(s.AllowedMedia) > 0 {
		medias = strings.Join(s.AllowedMedia, "|")
	}

	template := fmt.Sprintf(
		"/(?P<mark>%s)/(?P<storage>loc|rem)/(?P<size>%s)/(?P<path>%s.+)", proxyMark, sizes, medias)
	s.UrlExp, _ = regexp.Compile(template)
}

// makeCachePath generates cache path from resized image
func (s *Settings) makeCachePath() {
	var subPath string
	var cacheImageName string

	pathParts := strings.Split(s.Context.Path, "/")
	lastIndex := len(pathParts) - 1
	imageData := strings.Split(pathParts[lastIndex], ".")
	imageName, imageFormat := imageData[0], strings.ToLower(imageData[1])
	s.Context.Format = imageFormat

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

	s.Context.CachePath, _ = url.QueryUnescape(fmt.Sprintf(
		"%s/%s/%s", s.CacheDir, subPath, cacheImageName))

	if s.Context.Query != "" {
		s.Context.CachePath = fmt.Sprintf(
			"%s?%s", s.Context.CachePath, s.Context.Query)
	}
}

func fileExists(s *Settings) (string, error) {
	var filePath string
	var err error

	debug("Trying to find local image")
	s.Context.Path, _ = url.QueryUnescape(s.Context.Path)

	if len(s.Directories) > 0 {
		for _, dir := range s.Directories {
			filePath = path.Join("/", dir, s.Context.Path)
			if _, err = os.Stat(filePath); err == nil {
				return filePath, nil
			}
		}
		return "", err
	}

	filePath = path.Join("/", s.Context.Path)

	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		return "", err
	}

	return filePath, nil

}

// getLocalImage fetches original image from file system
func getLocalImage(s *Settings) ([]byte, error) {
	var image []byte
	var err error

	filePath, err := fileExists(s)
	if err != nil {
		return image, err
	}

	file, err := os.Open(filePath)
	defer file.Close()

	if err != nil {
		return image, err
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
func getRemoteImage(s *Settings, url string, isNode bool) ([]byte, error) {
	var image []byte
	var client = &http.Client{}

	debug("Trying to fetch remote image: %s", url)

	req, _ := http.NewRequest("GET", url, nil)

	if isNode {
		req.Header.Set(ONLY_CACHE_HEADER, "true")

		if s.Options.Webp {
			req.Header.Set("Accept", WEBP_HEADER)
		}
	}

	resp, err := client.Do(req)
	defer resp.Body.Close()

	if err != nil {
		return image, err
	}

	if resp.StatusCode != http.StatusOK {
		return image, errors.New("Not found")
	}

	image, err = ioutil.ReadAll(resp.Body)

	return image, nil
}

func checkCache(s *Settings) ([]byte, error) {
	var c *cache.Cache
	var image []byte
	var err error

	debug("Get from cache, key: %s", s.Context.CachePath)
	if image, err = c.Get(s.Context.CachePath); err == nil {
		return image, nil
	}

	if len(s.Nodes) > 0 && !s.Context.OnlyCache {
		debug("Checking other nodes")
		if image, err = checkNodes(s); err == nil {
			return image, nil
		}
	}

	debug("Image not found")
	return image, err
}

func checkNodes(s *Settings) ([]byte, error) {
	var image []byte
	var err error

	for _, node := range s.Nodes {
		reqUrl := fmt.Sprintf("%s://%s%s", s.Scheme, node, s.Context.RequestURI)
		if image, err = getRemoteImage(s, reqUrl, true); err == nil {
			debug("Found at node: %s", node)
			return image, nil
		}
	}

	return image, errors.New("No one node has the image")
}

// getOrCreateImage check cache path for requested image
// if image doesn't exist - creates it
func getOrCreateImage(sett Settings) []byte {

	var c *cache.Cache
	var image []byte
	var err error

	if !sett.Context.NoCache {
		if image, err = checkCache(&sett); err == nil {
			return image
		}
	}

	switch sett.Context.Storage {
	case "loc":
		image, err = getLocalImage(&sett)
		if err != nil {
			warning("Can't get orig local file - %s, reason - %s", sett.Context.Path, err)
			return image
		}

	case "rem":
		imgUrl := fmt.Sprintf("%s://%s", sett.Scheme, sett.Context.Path)
		image, err = getRemoteImage(&sett, imgUrl, false)
		if err != nil {
			warning("Can't get orig remote file - %s, reason - %s", sett.Context.Path, err)
			return image
		}
	}

	debug("Check image format")
	if !stringExists(sett.Context.Format, supportedFormats) {
		err = c.Set(sett.Context.CachePath, image)
		if err != nil {
			warning("Can't set cache, reason - %s", err)
		}
		return image
	}

	debug("Processing image")
	buf, err := vips.Resize(image, sett.Options)
	if err != nil {
		warning("Can't resize image, reason - %s", err)
	}

	debug("Set to cache, key: %s", sett.Context.CachePath)
	err = c.Set(sett.Context.CachePath, buf)
	if err != nil {
		warning("Can't set cache, reason - %s", err)
	}

	return buf
}

func stringExists(str string, list []string) bool {
	for _, el := range list {
		if el == str {
			return true
		}
	}
	return false
}

func parseVars(req *http.Request) map[string]string {
	params := map[string]string{"query": req.URL.RawQuery}
	match := settings.UrlExp.FindStringSubmatch(req.URL.Path)

	for i, name := range settings.UrlExp.SubexpNames() {
		params[name] = match[i]
	}

	return params
}

func fetchImage(rw http.ResponseWriter, req *http.Request) {
	acceptedTypes := strings.Split(req.Header.Get("Accept"), ",")
	noCacheKey := req.Header.Get(NO_CACHE_HEADER)
	onlyCache := req.Header.Get(ONLY_CACHE_HEADER)
	params := parseVars(req)
	sizes := strings.Split(params["size"], "x")
	sett := settings

	var resultImage []byte
	var err error

	sett.Options.Gravity = vips.CENTRE
	if crop := req.FormValue("crop"); crop != "" {
		for _, g := range strings.Split(crop, ",") {
			if v, ok := Crop[g]; ok {
				sett.Options.Gravity = sett.Options.Gravity | v
			}
		}
	}

	if q := req.FormValue("q"); q != "" {
		sett.Options.Quality, _ = strconv.Atoi(q)
	}

	sett.Options.Webp = stringExists(WEBP_HEADER, acceptedTypes)
	sett.Options.Width, _ = strconv.Atoi(sizes[0])
	sett.Options.Height, _ = strconv.Atoi(sizes[1])

	sett.Context.NoCache = sett.NoCacheKey != "" && sett.NoCacheKey == noCacheKey
	sett.Context.RequestURI = req.RequestURI
	sett.Context.Storage = params["storage"]
	sett.Context.Path = params["path"]
	sett.Context.Query = params["query"]

	sett.makeCachePath()

	ChanPool <- 1

	if onlyCache != "" {
		sett.Context.OnlyCache = true
		resultImage, err = checkCache(&sett)

		if err != nil {
			http.NotFound(rw, req)
		} else {
			rw.Write(resultImage)
		}

	} else {
		resultImage = getOrCreateImage(sett)
		contentLength := len(resultImage)

		if contentLength == 0 {
			debug("Content length 0")
			http.NotFound(rw, req)
		}

		rw.Header().Set("Content-Length", strconv.Itoa(contentLength))
		rw.Write(resultImage)
	}

	<-ChanPool
}

func init() {
	flag.Parse()
	settings.loadSettings()

	if os.Getenv("DEBUG_ENABLED") != "" {
		DEBUG = true
	}

	if os.Getenv("WARNING_ENABLED") != "" {
		WARNING = true
	}

	pool_size, err := strconv.Atoi(os.Getenv("IMGW_POOL_SIZE"))
	if err != nil {
		debug("Making channel with default size")
		ChanPool = make(chan int, DEFAULT_POOL_SIZE)
	} else {
		debug("Making channel, size %d", pool_size)
		ChanPool = make(chan int, pool_size)
	}
}

func main() {
	r := new(RegexpHandler)
	r.HandleFunc(settings.UrlExp, fetchImage)

	log.Printf("ImgWizard started on http://%s", settings.ListenAddr)
	http.ListenAndServe(settings.ListenAddr, r)
}

func debug(s string, args ...interface{}) {
	if !DEBUG {
		return
	}
	log.Printf(s+"\n", args...)
}

func warning(s string, args ...interface{}) {
	if !WARNING {
		return
	}
	log.Printf(s+"\n", args...)
}
