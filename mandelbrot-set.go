package main

import (
	"errors"
	"fmt"
	"golang.org/x/net/netutil"
	"image"
	"image/color"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"math/cmplx"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxIterations = 64
const connectionsCount = 20

var resolutions = map[string]uint64{
	"small":  64,
	"medium": 512,
	"big":    2048,
	"ultra":  4096,
}

type params struct {
	x    float64
	y    float64
	zoom uint64
	res  uint64
}

type heavyRequest struct {
	params  params
	channel chan *image.Gray
}

type mandelbrot struct {
	cacheDir string
	queue    chan heavyRequest
}

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Listen: %v\n", err)
	}
	l = netutil.LimitListener(l, connectionsCount)

	cacheDir, errCache := ioutil.TempDir("./", "cache")
	if errCache != nil {
		log.Fatalf("failed to create a deirectory to store the images: %v\n", errCache)
	}
	defer os.RemoveAll(cacheDir)
	heavyRequests := make(chan heavyRequest)
	mandelbrot := mandelbrot{cacheDir, heavyRequests}

	go mandelbrot.heavyRequestsProcessor()
	http.HandleFunc("/", mandelbrot.handler)
	serverErr := http.Serve(l, nil)
	if serverErr != nil {
		close(heavyRequests)
		log.Fatalf("failed to start the server: %v\n", serverErr)
	}
}

func (m mandelbrot) heavyRequestsProcessor() {
	for heavyRequest := range m.queue {
		time.Sleep(20 * time.Second)
		heavyRequest.channel <- m.calculateImage(heavyRequest.params)
	}
}

func (m mandelbrot) handler(w http.ResponseWriter, r *http.Request) {
	params, errParse := parseParams(r.URL.Query())
	if errParse != nil {
		log.Println("400:", errParse)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("400: %v", errParse)))
		return
	}

	w.Header().Set("Content-Type", "image/png")
	imgPath := m.imagePath(params)
	_, errCache := os.Stat(imgPath)
	if errCache == nil {
		content, errRead := ioutil.ReadFile(imgPath)
		if errRead == nil {
			w.Write(content)
			return
		}

		log.Println("failed to read a cached image:", imgPath, errRead)
	}

	if (params.res == resolutions["small"]) || (params.res == resolutions["medium"]) {
		img := m.calculateImage(params)
		errEncode := png.Encode(w, img)
		if errEncode != nil {
			log.Println("failed to encode an image:", errEncode)
		}
		m.storeImage(imgPath, img)
		return
	}

	channel := make(chan *image.Gray)
	m.queue <- heavyRequest{params, channel}
	bigImg := <-channel
	errEncodeBig := png.Encode(w, bigImg)
	if errEncodeBig != nil {
		log.Println("failed to encode an image:", errEncodeBig)
	}
	m.storeImage(imgPath, bigImg)
}

func (m mandelbrot) storeImage(imgPath string, img *image.Gray) {
	info, errStat := os.Lstat(m.cacheDir)
	if errStat != nil {
		log.Fatalf("failed to get the cache dir stats: %v\n", errStat)
	}
	if info.Size() > int64(15*math.Pow(2, 30)) {
		log.Println("not enough disk space to cache an image")
		return
	}

	f, errCreateFile := os.Create(imgPath)
	if errCreateFile != nil {
		log.Println("failed to create an image file", imgPath, errCreateFile)
	}
	defer f.Close()

	errWrite := png.Encode(f, img)
	if errWrite != nil {
		log.Println("failed to store an image", imgPath, errWrite)
	}
}

func parseParams(q url.Values) (params, error) {
	x, errX := parseX(q.Get("x"))
	if errX != nil {
		return params{}, errX
	}

	y, errY := parseY(q.Get("y"))
	if errY != nil {
		return params{}, errY
	}

	zoom, errZoom := parseZoom(q.Get("zoom"))
	if errZoom != nil {
		return params{}, errZoom
	}

	res, errRes := parseRes(q.Get("res"))
	if errRes != nil {
		return params{}, errRes
	}

	return params{x, y, zoom, res}, nil
}

func parseX(sX string) (float64, error) {
	x, err := strconv.ParseFloat(sX, 64)
	if err != nil {
		return x, errors.New("invalid x")
	}
	return x, err
}

func parseY(sY string) (float64, error) {
	y, err := strconv.ParseFloat(sY, 64)
	if err != nil {
		return y, errors.New("invalid y")
	}
	return y, err
}

func parseZoom(sZoom string) (uint64, error) {
	zoom, errZoom := strconv.ParseUint(sZoom, 10, 64)
	if errZoom != nil {
		return zoom, errors.New("invalid zoom")
	}
	if zoom < 1 {
		return zoom, errors.New("zoom must be at least 1")
	}
	return zoom, errZoom

}

func parseRes(sRes string) (uint64, error) {
	res, ok := resolutions[sRes]
	if !ok {
		return res, errors.New("invalid res")
	}
	return res, nil
}

func (m mandelbrot) calculateImage(params params) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, int(params.res), int(params.res)))

	delta := float64(params.res) / float64((2 * params.zoom))
	left := params.x - delta
	top := params.y + delta
	step := 1 / float64(params.zoom)
	for y := 0; y < int(params.res); y++ {
		for x := 0; x < int(params.res); x++ {
			if bailOut(complex(left+float64(x)*step, top-float64(y)*step)) {
				img.Set(x, y, color.Gray{255})
			} else {
				img.Set(x, y, color.Gray{0})
			}
		}
	}
	return img
}

func bailOut(c complex128) bool {
	z := c
	for i := 0; i < maxIterations; i++ {
		if cmplx.Abs(z) > 2 {
			return false
		}
		z = cmplx.Pow(z, 2) + c
	}
	return true
}

func (m mandelbrot) imagePath(params params) string {
	imgNameParts := []string{
		strconv.FormatFloat(params.x, 'E', -1, 64),
		strconv.FormatFloat(params.y, 'E', -1, 64),
		strconv.FormatUint(params.zoom, 10),
		strconv.FormatUint(params.res, 10),
	}
	imgName := fmt.Sprintf("%s.png", strings.Join(imgNameParts, "-"))
	return filepath.Join(m.cacheDir, imgName)
}
