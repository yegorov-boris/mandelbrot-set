package main

import (
	"net/http"
	"fmt"
	"log"
	"net"
	"golang.org/x/net/netutil"
	"time"
	"strconv"
	"errors"
)

const connectionsCount = 2
const sleepBeforeRestart = 5 * time.Second
var resolutions = map[string]int{
	"small": 64,
	"medium": 512,
	"big": 2048,
	"ultra": 4096,
}

type params struct {
	x float64
	y float64
	zoom uint64
	res int
}

func main() {
	startServer()
}

func startServer() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Listen: %v\n", err)
	}
	l = netutil.LimitListener(l, connectionsCount)
	defer l.Close()

	http.HandleFunc("/", handler)
	serverErr := http.Serve(l, nil)
	if serverErr != nil {
		log.Printf("Server failed: %v\n", serverErr)
		log.Println("Restarting")
		time.Sleep(sleepBeforeRestart)
		startServer()
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	x, errX := parseX(q.Get("x"))
	if errX!=nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("400: %v", errX)))
		return
	}

	y, errY := parseY(q.Get("y"))
	if errY!=nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("400: %v", errY)))
		return
	}

	zoom, errZoom := parseZoom(q.Get("zoom"))
	if errZoom!=nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("400: %v", errZoom)))
		return
	}

	res, errRes := parseRes(q.Get("res"))
	if errRes!=nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("400: %v", errRes)))
		return
	}

	fmt.Fprintf(w, foo(params{x, y, zoom, res}))
}

func parseX(sX string) (float64, error) {
	x, err := strconv.ParseFloat(sX, 64)
	if err!=nil {
		return x, errors.New("invalid x")
	}
	return x, err
}

func parseY(sY string) (float64, error) {
	y, err := strconv.ParseFloat(sY, 64)
	if err!=nil {
		return y, errors.New("invalid y")
	}
	return y, err
}

func parseZoom(sZoom string) (uint64, error) {
	zoom, errZoom := strconv.ParseUint(sZoom, 10, 64)
	if errZoom!= nil {
		return zoom, errors.New("invalid zoom")
	}
	if zoom<1 {
		return zoom, errors.New("zoom must be at least 1")
	}
	return zoom, errZoom


}

func parseRes(sRes string) (int, error) {
	res, ok := resolutions[sRes]
	if !ok {
		return res, errors.New("invalid res")
	}
	return res, nil
}

func foo(params params) string {
	return "foo"
}
