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
)

var options vips.Options

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

func Index(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	fmt.Fprint(rw, `
				<html>
					<body>
					Hello, here will be an image!<br/>
					<img src="/images/320x240/car.jpg">
					</body>
				</html>`)

}

func FetchLocalImage(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	sizes := strings.Split(params["size"], "x")
	imgPath := params["path"]
	imgWidth, _ := strconv.Atoi(sizes[0])
	imgHeight, _ := strconv.Atoi(sizes[1])

	options.Width = imgWidth
	options.Height = imgHeight

	file, err := os.Open(path.Join("/", imgPath))
	if err != nil {
		log.Println(err)
		file, _ = os.Open(LOCAL_404_THUMB)
	}
	defer file.Close()

	info, _ := file.Stat()
	image := make([]byte, info.Size())

	_, err = file.Read(image)
	if err != nil {
		log.Println(os.Stderr, err)
	}

	buf, err := vips.Resize(image, options)
	if err != nil {
		log.Println(os.Stderr, err)
	}

	rw.Header().Set("Content-Type", "image/jpg")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Write(buf)
}

func FetchRemoteImage(rw http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	sizes := strings.Split(params["size"], "x")
	imgPath := params["path"]
	imgWidth, _ := strconv.Atoi(sizes[0])
	imgHeight, _ := strconv.Atoi(sizes[1])

	options.Width = imgWidth
	options.Height = imgHeight

	imgUrl := fmt.Sprintf("%s://%s", SCHEME, imgPath)
	resp, err := http.Get(imgUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	image, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	buf, err := vips.Resize(image, options)
	if err != nil {
		log.Println(os.Stderr, err)
		return
	}

	rw.Header().Set("Content-Type", "image/jpg")
	rw.Header().Set("Access-Control-Allow-Origin", "*")
	rw.Write(buf)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", Index)
	r.HandleFunc("/images/loc/{size:[0-9]*x[0-9]*}/{path:.+}", FetchLocalImage).Methods("GET")
	r.HandleFunc("/images/rem/{size:[0-9]*x[0-9]*}/{path:.+}", FetchRemoteImage).Methods("GET")

	http.ListenAndServe(":8070", r)
}
