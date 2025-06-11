package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/static"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
)

func doServe(params templates.Params) error {
	var (
		buf       bytes.Buffer
		mainBytes []byte
	)
	err := templates.Main(params).Render(context.Background(), &buf)
	if err != nil {
		return err
	}
	mainBytes = buf.Bytes()

	qual := func(endpoint string) string {
		return strings.Replace(endpoint, "/", params.BasePath+"/", 1)
	}

	log.Println("Enumerating imgFS")
	err = fs.WalkDir(imgFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dirInd := "f"
		if d.IsDir() {
			dirInd = "d"
		}
		log.Printf("[%s] %s\n", dirInd, path)
		return nil
	})
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle(qual("GET /img/"), loggf(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		path = strings.TrimPrefix(path, qual("/"))
		log.Println("Serving image from FS with path ", path)
		data, err := imgFS.ReadFile(path)
		if err != nil {
			http.Error(w, "Error: "+err.Error(), 500)
			log.Printf("Error reading file '%s': %v\n", path, err)
			return
		}
		if strings.HasSuffix(path, "svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		_, _ = w.Write(data)
	}))
	indexHandler := loggf(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err := w.Write(mainBytes)
		if err != nil {
			w.WriteHeader(500)
		}
	})
	mux.Handle(qual("GET /index.html"), indexHandler)
	mux.Handle(qual("GET /{$}"), indexHandler)
	mux.Handle(qual("GET /main.css"), loggf(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, err := w.Write(static.MainCSS)
		if err != nil {
			w.WriteHeader(500)
		}
	}))
	fmt.Println("Serving docs site")
	return http.ListenAndServe(":8080", mux) //nolint:gosec // This is just a local server, no timeout needed.
}

func logg(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingWriter{
			ResponseWriter: w,
		}
		start := time.Now()
		next.ServeHTTP(lw, r)
		taken := time.Since(start)
		log.Printf("[%d] %s %s %s %d\n", lw.status(), r.Method, r.URL.Path, taken.Round(time.Microsecond).String(), lw.written())
	}
}

func loggf(next http.HandlerFunc) http.HandlerFunc {
	return logg(next)
}

type loggingWriter struct {
	http.ResponseWriter
	savedStatus  int
	bytesWritten int
}

func (lw *loggingWriter) Write(bytes []byte) (int, error) {
	written, err := lw.ResponseWriter.Write(bytes)
	lw.bytesWritten += written
	if err != nil {
		log.Println("Error writing to client:", err)
	}
	return written, err
}

func (lw *loggingWriter) WriteHeader(status int) {
	lw.ResponseWriter.WriteHeader(status)
	lw.savedStatus = status
}

func (lw *loggingWriter) status() int {
	if lw.savedStatus == 0 {
		return 200
	}
	return lw.savedStatus
}

func (lw *loggingWriter) written() int {
	return lw.bytesWritten
}
