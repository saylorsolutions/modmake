package main

import (
	"log"
	"net/http"
)

func main() {
	srv := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			log.Println("Received PING")
			_, _ = writer.Write([]byte("PONG"))
		}),
	}

	log.Println("Starting server")
	err := srv.ListenAndServe()
	log.Println(err)
}
