package main

import (
	"net/http"
	"fmt"
	"log"
	"net"
	"golang.org/x/net/netutil"
	"time"
)

const connectionsCount = 2
const sleepBeforeRestart = 5 * time.Second

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
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}
