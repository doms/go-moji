package main

import (
	"log"
	"net/http"
	"os"

	"github.com/doms/go-moji/emoji"
)

func getPort() string {
	p := os.Getenv("PORT")
	if p != "" {
		return ":" + p
	}
	return ":8080"
}

func main() {
	// serve stuff from static
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", emoji.IndexHandler)
	http.HandleFunc("/fetch-skin-tones", emoji.FetchSkinTonesHandler)

	// listen on port
	log.Fatal(http.ListenAndServe(getPort(), nil))
}
