package main

import (
	"net/http"
	"fmt"
	"strconv"
	"errors"
	"net"
	"log"
	"golang.org/x/net/netutil"
	"net/url"
	"time"
)

const connectionsCount = 20
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

type heavyRequest struct {
	params params
	channel chan string
}

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Listen: %v\n", err)
	}
	l = netutil.LimitListener(l, connectionsCount)
	defer l.Close()

	heavyRequests := make(chan heavyRequest)
	go heavyRequestsProcessor(heavyRequests)
	http.HandleFunc("/", createHandler(heavyRequests))
	serverErr := http.Serve(l, nil)
	if serverErr!= nil {
		close(heavyRequests)
		log.Fatal(serverErr)
	}
}

func heavyRequestsProcessor(queue chan heavyRequest) {
	for heavyRequest := range queue {
		time.Sleep(20 * time.Second)
		heavyRequest.channel <- foo(heavyRequest.params)
	}
}

func createHandler(queue chan heavyRequest) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		params, err := parseParams(r.URL.Query())
		if err!=nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("400: %v", err)))
			return
		}

		if (params.res == resolutions["small"]) || (params.res == resolutions["medium"]) {
			fmt.Fprintf(w, foo(params))
			return
		}

		channel := make(chan string)
		queue <- heavyRequest{params, channel}
		fmt.Fprintf(w, <- channel)
	}
}

func parseParams(q url.Values) (params, error) {
	x, errX := parseX(q.Get("x"))
	if errX!=nil {
		return params{}, errX
	}

	y, errY := parseY(q.Get("y"))
	if errY!=nil {
		return params{}, errY
	}

	zoom, errZoom := parseZoom(q.Get("zoom"))
	if errZoom!=nil {
		return params{}, errZoom
	}

	res, errRes := parseRes(q.Get("res"))
	if errRes!=nil {
		return params{}, errRes
	}

	return params{x, y, zoom, res}, nil
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
