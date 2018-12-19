package main

import (
	"log"
	"net/http"

	"github.com/doms/go-moji/emoji"
)

func main() {
	// serve stuff from static
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", emoji.IndexHandler)
	http.HandleFunc("/fetch-skin-tones", emoji.FetchSkinTonesHandler)

	// listen on port
	log.Println("Listening on :4567...")
	log.Fatal(http.ListenAndServe(":4567", nil))
}
