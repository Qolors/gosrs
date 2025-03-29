package server

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	s http.Server
)

func fileHandler(w http.ResponseWriter, r *http.Request) {
	fs, err := os.Open("serve/overview.html")

	if err != nil {
		fmt.Print(err.Error())

	}

	defer fs.Close()

	info, err := fs.Stat()

	if err != nil {
		fmt.Print(err.Error())

	}

	http.ServeContent(w, r, info.Name(), info.ModTime(), fs)
}

func StartAndListen() {

	s := &http.Server{
		Addr:           ":8080",
		ReadTimeout:    10 * time.Second,
		Handler:        http.HandlerFunc(fileHandler),
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	err := s.ListenAndServe()

	if err == http.ErrServerClosed {

		fmt.Sprintln("Server Closed")

	}
}

func CloseServer() {

	s.Close()
}
