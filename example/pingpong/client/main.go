package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	log.Println("Starting ping-pong loop")
	for {
		log.Println("PING")
		resp, err := http.Get("http://localhost:8080/ping")
		if err != nil {
			panic(err)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		log.Println(string(data))
		time.Sleep(time.Second)
	}
}
