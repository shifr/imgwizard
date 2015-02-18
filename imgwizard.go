package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/daddye/vips"
	"github.com/gorilla/mux"
)

const (
	LOCAL_404_THUMB = "/tmp/default.jpg"
	SCHEME          = "http"
	CACHE_DIR       = "/tmp"
)

var options vips.Options

type Context struct {
	Path      string
	CachePath string
	Storage   string
	Width     int
	Height    int
}

func (c *Context) makeCachePath() {
	var subPath string
	pathParts := strings.Split(c.Path, "/")
	lastIndex := len(pathParts) - 1
	imageData := strings.Split(pathParts[lastIndex], ".")
	imageName, imageFormat := imageData[0], imageData[1]
	cacheImageName := fmt.Sprintf("%s_%dx%d.%s", imageName, c.Width, c.Height, imageFormat)

	switch c.Storage {
	case "loc":
		subPath = strings.Join(pathParts[:lastIndex], "/")
	case "rem":
		subPath = strings.Join(pathParts[1:lastIndex], "/")
	}

	c.CachePath = fmt.Sprintf("%s/%s/%s", CACHE_DIR, subPath, cacheImageName)
}

func init() {
	options = vips.Options{
		Crop:         true,
		Enlarge:      true,
		Extend:       vips.EXTEND_WHITE,
		Interpolator: vips.BILINEAR,
		Gravity:      vips.CENTRE,
		Quality:      80,
	}
}

func fetchImage(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	sizes := strings.Split(params["size"], "x")

	c := new(Context)
	c.Storage = params["storage"]
	c.Path = params["path"]
	c.Width, _ = strconv.Atoi(sizes[0])
	c.Height, _ = strconv.Atoi(sizes[1])

	resultImage := getOrCreateImage(c)

	rw.Header().Set("Content-Type", "image/jpg")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Write(resultImage)
}

func getOrCreateImage(c *Context) []byte {
	c.makeCachePath()

	var file *os.File
	var info os.FileInfo
	var image []byte
	var resp *http.Response
	var err error

	defer file.Close()

	if file, err = os.Open(c.CachePath); err == nil {

		info, _ = file.Stat()
		image = make([]byte, info.Size())

		_, err = file.Read(image)
		if err != nil {
			log.Println("Can't read cached file, reason - ", err)
		}

		return image
	}

	switch c.Storage {
	case "loc":
		file, err = os.Open(path.Join("/", c.Path))
		if err != nil {
			log.Println("Can't read orig file, reason - ", err)
			file, _ = os.Open(LOCAL_404_THUMB)
		}

		info, _ = file.Stat()
		image = make([]byte, info.Size())

		_, err = file.Read(image)
		if err != nil {
			log.Println("Can't read file to image, reason - ", err)
		}
	case "rem":
		imgUrl := fmt.Sprintf("%s://%s", SCHEME, c.Path)

		resp, err = http.Get(imgUrl)
		if err != nil {
			log.Println("Can't fetch image from url, reason - ", err)
		}
		defer resp.Body.Close()

		image, _ = ioutil.ReadAll(resp.Body)
	}

	options.Width = c.Width
	options.Height = c.Height

	buf, err := vips.Resize(image, options)
	if err != nil {
		log.Println("Can't resize image, reason - ", err)
	}
	err = os.MkdirAll(path.Dir(c.CachePath), 0777)
	if err != nil {
		log.Println("Can't make dir, reason - ", err)
	}
	err = ioutil.WriteFile(c.CachePath, buf, 0666)
	if err != nil {
		log.Println("Can't write file, reason - ", err)
	}

	return buf
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/images/{storage:loc|rem}/{size:[0-9]*x[0-9]*}/{path:.+}", fetchImage).Methods("GET")

	http.ListenAndServe(":8070", r)
}
