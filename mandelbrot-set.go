package main

import (
	"net/http"
	"fmt"
	"log"
	"net"
	"golang.org/x/net/netutil"
)

const connectionsCount = 2

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Listen: %v", err)
	}
	l = netutil.LimitListener(l, connectionsCount)
	defer l.Close()

	http.HandleFunc("/", handler)
	log.Fatal(http.Serve(l, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}
