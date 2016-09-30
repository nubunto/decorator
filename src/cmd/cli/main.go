package main

import (
	"bytes"
	"client"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type proxy struct {
	client.Client
}

func (h proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	res, err := h.Client.Do(req)
	if err != nil {
		// log?
		log.Println(err)
		return
	}
	if _, err := io.Copy(w, res.Body); err != nil {
		// log? retry? wtf?
		log.Println(err)
		return
	}
	return
}

func main() {
	final := client.Decorate(
		&http.Client{},
		client.Logging(log.New(os.Stdout, "client: ", log.LstdFlags)),
		client.Proxy(client.Match(func(b []byte) bool {
			return bytes.Equal(b, []byte("hello world"))
		}, "http://localhost:8090", "http://localhost:8091")),
		client.FaultTolerance(5, time.Second),
	)
	h := proxy{final}
	http.ListenAndServe(":8080", h)
}
