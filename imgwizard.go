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
	Width      int
	Height     int
	Path       string
	RequestURI string
	CachePath  string
	Storage    string
	Query      string

	Options vips.Options
}

type Settings struct {
	Scheme       string
	AllowedSizes []string
	AllowedMedia []string
	Directories  []string
	Nodes        []string
	UrlExp       *regexp.Regexp

	Context Context
}

const (
	VERSION                  = 1.3
	DEFAULT_POOL_SIZE        = 100000
	WEBP_HEADER              = "image/webp"
	ONLY_CACHE_HEADER        = "X-Cache-Only"
	NO_CACHE_HEADER          = "X-No-Cache"
	CACHE_DESTINATION_HEADER = "X-Cache-Destination"
)

var (
	DEBUG           = false
	WARNING         = false
	DEFAULT_QUALITY = 80

	settings           Settings
	Cache              *cache.Cache
	Options            vips.Options
	ChanPool           chan int
	ListenAddr         string
	AllowedMedia       string
	AllowedSizes       string
	CacheDir           string
	S3BucketName       string
	AzureContainerName string
	Default404         string
	DirsToSearch       string
	Mark               string
	NoCacheKey         string
	Nodes              string
	Quality            int

	Crop = map[string]vips.Gravity{
		"top":    vips.NORTH,
		"right":  vips.EAST,
		"bottom": vips.SOUTH,
		"left":   vips.WEST,
	}
	ResizableImageTypes = []string{"image/jpeg", "image/png"}
)

// makeCachePath generates cache path for resized image
func (c *Context) makeCachePath() {
	var subPath string
	var cacheImageName string
	var imageFormat string

	pathParts := strings.Split(c.Path, "/")
	lastIndex := len(pathParts) - 1
	imageName := pathParts[lastIndex]
	imageNameParts := strings.Split(imageName, ".")

	if len(imageNameParts) > 1 {
		lastNameIndex := len(imageNameParts) - 1
		imageName = strings.Join(imageNameParts[:lastNameIndex], ".")
		imageFormat = imageNameParts[lastNameIndex]
	}

	if c.Options.Webp {
		cacheImageName = fmt.Sprintf(
			"%s_%dx%d_webp", imageName, c.Options.Width, c.Options.Height)
	} else {
		cacheImageName = fmt.Sprintf(
			"%s_%dx%d", imageName, c.Options.Width, c.Options.Height)
	}

	if imageFormat != "" {
		cacheImageName = fmt.Sprintf("%s.%s", cacheImageName, imageFormat)
	}

	subPath = strings.Join(pathParts[:lastIndex], "/")

	if S3BucketName != "" || AzureContainerName != "" {
		c.CachePath, _ = url.QueryUnescape(fmt.Sprintf(
			"%s/%s", subPath, cacheImageName))
	} else {
		c.CachePath, _ = url.QueryUnescape(fmt.Sprintf(
			"%s/%s/%s", CacheDir, subPath, cacheImageName))
	}

	if c.Query != "" {
		c.CachePath = fmt.Sprintf(
			"%s?%s", c.CachePath, c.Query)
	}
}

func (c *Context) Fill(req *http.Request) {
	acceptedTypes := strings.Split(req.Header.Get("Accept"), ",")
	noCacheKey := req.Header.Get(NO_CACHE_HEADER)
	onlyCacheHeader := req.Header.Get(ONLY_CACHE_HEADER)
	cachePath := req.Header.Get(CACHE_DESTINATION_HEADER)
	params := parseVars(req)
	sizes := strings.Split(params["size"], "x")
	c.Options = Options
	c.Options.Gravity = vips.CENTRE

	if crop := req.FormValue("crop"); crop != "" {
		for _, g := range strings.Split(crop, ",") {
			if v, ok := Crop[g]; ok {
				c.Options.Gravity = c.Options.Gravity | v
			}
		}
	}

	if q := req.FormValue("q"); q != "" {
		c.Options.Quality, _ = strconv.Atoi(q)
	}

	c.Options.Webp = stringExists(WEBP_HEADER, acceptedTypes)
	c.Options.Width, _ = strconv.Atoi(sizes[0])
	c.Options.Height, _ = strconv.Atoi(sizes[1])

	c.NoCache = NoCacheKey != "" && NoCacheKey == noCacheKey
	c.OnlyCache = onlyCacheHeader != ""
	c.RequestURI = req.RequestURI
	c.Storage = params["storage"]
	c.Path = params["path"]
	c.Query = params["query"]

	if cachePath != "" {
		c.CachePath = cachePath
		return
	}

	c.makeCachePath()
}

// loadSettings loads settings from command-line
func (s *Settings) loadSettings() {

	s.Scheme = "http"
	s.AllowedSizes = nil
	s.AllowedMedia = nil

	//defaults for vips
	Options.Crop = true
	Options.Enlarge = true
	Options.Extend = vips.EXTEND_WHITE
	Options.Interpolator = vips.BILINEAR

	var sizes = "[0-9]*x[0-9]*"
	var medias = ""

	if AllowedMedia != "" {
		s.AllowedMedia = strings.Split(AllowedMedia, ",")
	}

	if AllowedSizes != "" {
		s.AllowedSizes = strings.Split(AllowedSizes, ",")
	}

	if DirsToSearch != "" {
		s.Directories = strings.Split(DirsToSearch, ",")
	}

	if Nodes != "" {
		s.Nodes = strings.Split(Nodes, ",")
	}

	if Quality != 0 {
		DEFAULT_QUALITY = Quality
	}
	Options.Quality = DEFAULT_QUALITY

	if len(s.AllowedSizes) > 0 {
		sizes = strings.Join(s.AllowedSizes, "|")
	}

	if len(s.AllowedMedia) > 0 {
		medias = strings.Join(s.AllowedMedia, "|")
	}

	template := fmt.Sprintf(
		"/(?P<mark>%s)/(?P<storage>loc|rem)/(?P<size>%s)/(?P<path>((%s)(.+)))",
		Mark, sizes, medias)
	debug("Template %s", template)
	s.UrlExp, _ = regexp.Compile(template)
}

func fileExists(ctx *Context) (string, error) {
	var filePath string
	var err error

	debug("Trying to find local image")
	ctx.Path, _ = url.QueryUnescape(ctx.Path)

	if len(settings.Directories) > 0 {
		for _, dir := range settings.Directories {
			filePath = path.Join("/", dir, ctx.Path)
			if _, err = os.Stat(filePath); err == nil {
				return filePath, nil
			}
		}
		return "", err
	}

	filePath = path.Join("/", ctx.Path)

	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		return "", err
	}

	return filePath, nil

}

// getLocalImage fetches original image from file system
func getLocalImage(ctx *Context, def bool) ([]byte, error) {
	var image []byte
	var err error
	var filePath string

	if def {
		filePath = Default404
	} else {
		filePath, err = fileExists(ctx)
		if err != nil {
			return image, err
		}
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
func getRemoteImage(ctx *Context, url string, isNode bool) ([]byte, error) {
	var image []byte
	var client = &http.Client{}

	debug("Trying to fetch remote image: %s", url)

	req, _ := http.NewRequest("GET", url, nil)

	if isNode {
		req.Header.Set(ONLY_CACHE_HEADER, "true")

		if ctx.Options.Webp {
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

func checkCache(ctx *Context) ([]byte, error) {

	var image []byte
	var err error

	debug("Get from cache, key: %s", ctx.CachePath)
	if image, err = Cache.Get(ctx.CachePath); err == nil {
		return image, nil
	}

	if len(settings.Nodes) > 0 && !ctx.OnlyCache {
		debug("Checking other nodes")
		if image, err = checkNodes(ctx); err == nil {
			return image, nil
		}
	}

	debug("Image not found")
	return image, err
}

func checkNodes(ctx *Context) ([]byte, error) {
	var image []byte
	var err error

	for _, node := range settings.Nodes {
		reqUrl := fmt.Sprintf("%s://%s%s", settings.Scheme, node, ctx.RequestURI)
		if image, err = getRemoteImage(ctx, reqUrl, true); err == nil {
			debug("Found at node: %s", node)
			return image, nil
		}
	}

	return image, errors.New("No one node has the image")
}

// getOrCreateImage check cache path for requested image
// if image doesn't exist - creates it
func getOrCreateImage(ctx *Context) []byte {

	var image []byte
	var err error

	if !ctx.NoCache {
		if image, err = checkCache(ctx); err == nil {
			return image
		}
	}

	switch ctx.Storage {
	case "loc":
		image, err = getLocalImage(ctx, false)
		if err != nil {
			warning("Can't get orig local file - %s, reason - %s", ctx.Path, err)
			if Default404 != "" {
				image, err = getLocalImage(ctx, true)

				if err != nil {
					warning("Default 404 image was set but not found", Default404)
					return image
				}
			}
			return image
		}

	case "rem":
		imgUrl := fmt.Sprintf("%s://%s", settings.Scheme, ctx.Path)
		image, err = getRemoteImage(ctx, imgUrl, false)
		if err != nil {
			warning("Can't get orig remote file - %s, reason - %s", ctx.Path, err)
			if Default404 != "" {
				image, err = getLocalImage(ctx, true)

				if err != nil {
					warning("Default 404 image was set but not found", Default404)
					return image
				}
			}
			return image
		}
	}

	debug("Detecting image type...")
	iType := http.DetectContentType(image)

	if !stringExists(iType, ResizableImageTypes) {
		warning("Wizard resize doesn't support image type, returning original image")
		return image
	}

	debug("Processing image...")
	buf, err := vips.Resize(image, ctx.Options)
	if err != nil {
		warning("Can't resize image, reason - %s", err)

		err = Cache.Set(ctx.CachePath, image)
		if err != nil {
			warning("Can't set cache, reason - %s", err)
		}
		return image
	}

	debug("Set to cache, key: %s", ctx.CachePath)
	err = Cache.Set(ctx.CachePath, buf)
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
	ChanPool <- 1

	var resultImage []byte
	var err error

	context := Context{}
	context.Fill(req)

	if context.OnlyCache {
		resultImage, err = checkCache(&context)

		if err != nil {
			http.NotFound(rw, req)
		} else {
			rw.Write(resultImage)
		}

	} else {
		resultImage = getOrCreateImage(&context)
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
	log.SetOutput(os.Stdout)

	flag.StringVar(&ListenAddr, "l", "127.0.0.1:8070", "Address to listen on")
	flag.StringVar(&AllowedMedia, "m", "", "comma separated list of allowed media server hosts")
	flag.StringVar(&AllowedSizes, "s", "", "comma separated list of allowed sizes")
	flag.StringVar(&CacheDir, "c", "/tmp/imgwizard", "directory for cached files")
	flag.StringVar(&S3BucketName, "s3-b", "", "AWS S3 cache bucket name")
	flag.StringVar(&AzureContainerName, "az", "", "Microsoft Azure Storage container name")
	flag.StringVar(&Default404, "thumb", "", "path to default image if original not found")
	flag.StringVar(&DirsToSearch, "d", "", "comma separated list of directories to search requested file")
	flag.StringVar(&Mark, "mark", "images", "Mark for nginx")
	flag.StringVar(&NoCacheKey, "no-cache-key", "", "Secret key that must be equal X-No-Cache value from request header")
	flag.StringVar(&Nodes, "nodes", "", "Other imgwizard nodes to ask before process image")
	flag.IntVar(&Quality, "q", 0, "image quality after resize")

	if os.Getenv("DEBUG_ENABLED") != "" {
		DEBUG = true
		WARNING = true
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
	var err error

	flag.Parse()
	settings.loadSettings()

	Cache, err = cache.NewCache(S3BucketName, AzureContainerName)

	if err != nil {
		warning("Could not create cache object, reason - %s", err)
		return
	}

	r := new(RegexpHandler)
	r.HandleFunc(settings.UrlExp, fetchImage)

	log.Printf("ImgWizard started on http://%s", ListenAddr)
	http.ListenAndServe(ListenAddr, r)
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
