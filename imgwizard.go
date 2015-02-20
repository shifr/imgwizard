package main

import (
	"encoding/json"
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

type Context struct {
	Path      string
	CachePath string
	Storage   string
	Width     int
	Height    int
}

type Settings struct {
	Quality       int
	CacheDir      string
	Scheme        string
	Local404Thumb string
	AllowedSizes  []string
	AllowedMedia  []string
	UrlTemplate   string

	Context Context
	Options vips.Options
}

var settings Settings

func (s *Settings) loadSettings() {

	s.Options.Extend = vips.EXTEND_WHITE
	s.Options.Interpolator = vips.BILINEAR
	s.Options.Gravity = vips.CENTRE

	file, _ := ioutil.ReadFile("settings.json")

	err := json.Unmarshal(file, &s)
	if err != nil {
		log.Panic("Can't unmarshal settings, reason - ", err)
	}

	sizes := strings.Join(s.AllowedSizes, "|")
	s.UrlTemplate = fmt.Sprintf(
		"/images/{storage:loc|rem}/{size:%s}/{path:.+}", sizes)
}

func (s *Settings) makeCachePath() {
	var subPath string
	pathParts := strings.Split(s.Context.Path, "/")
	lastIndex := len(pathParts) - 1
	imageData := strings.Split(pathParts[lastIndex], ".")
	imageName, imageFormat := imageData[0], imageData[1]
	cacheImageName := fmt.Sprintf(
		"%s_%dx%d.%s", imageName, s.Options.Width, s.Options.Height, imageFormat)

	switch s.Context.Storage {
	case "loc":
		subPath = strings.Join(pathParts[:lastIndex], "/")
	case "rem":
		subPath = strings.Join(pathParts[1:lastIndex], "/")
	}

	s.Context.CachePath = fmt.Sprintf(
		"%s/%s/%s", s.CacheDir, subPath, cacheImageName)
}

func fetchImage(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	sizes := strings.Split(params["size"], "x")

	settings.Context.Storage = params["storage"]
	settings.Context.Path = params["path"]
	settings.Options.Width, _ = strconv.Atoi(sizes[0])
	settings.Options.Height, _ = strconv.Atoi(sizes[1])

	resultImage := getOrCreateImage()

	rw.Write(resultImage)
}

func getOrCreateImage() []byte {
	settings.makeCachePath()

	var file *os.File
	var info os.FileInfo
	var image []byte
	var resp *http.Response
	var err error

	defer file.Close()

	if file, err = os.Open(settings.Context.CachePath); err == nil {

		info, _ = file.Stat()
		image = make([]byte, info.Size())

		_, err = file.Read(image)
		if err != nil {
			log.Println("Can't read cached file, reason - ", err)
		}

		return image
	}

	switch settings.Context.Storage {
	case "loc":
		file, err = os.Open(path.Join("/", settings.Context.Path))
		if err != nil {
			log.Println("Can't read orig file, reason - ", err)
			file, _ = os.Open(settings.Local404Thumb)
		}

		info, _ = file.Stat()
		image = make([]byte, info.Size())

		_, err = file.Read(image)
		if err != nil {
			log.Println("Can't read file to image, reason - ", err)
		}
	case "rem":
		imgUrl := fmt.Sprintf("%s://%s", settings.Scheme, settings.Context.Path)

		resp, err = http.Get(imgUrl)
		if err != nil {
			log.Println("Can't fetch image from url, reason - ", err)
		}
		defer resp.Body.Close()

		image, _ = ioutil.ReadAll(resp.Body)
	}

	buf, err := vips.Resize(image, settings.Options)
	if err != nil {
		log.Println("Can't resize image, reason - ", err)
	}

	err = os.MkdirAll(path.Dir(settings.Context.CachePath), 0777)
	if err != nil {
		log.Println("Can't make dir, reason - ", err)
	}

	err = ioutil.WriteFile(settings.Context.CachePath, buf, 0666)
	if err != nil {
		log.Println("Can't write file, reason - ", err)
	}

	return buf
}

func init() {
	settings.loadSettings()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc(settings.UrlTemplate, fetchImage).Methods("GET")

	log.Println("ImgWizard started...")
	http.ListenAndServe(":8070", r)
}
