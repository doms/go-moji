package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var templates = template.Must(template.ParseGlob("templates/*"))

type emoji struct {
	Keywords         []string
	Char             string
	FitzpatrickScale bool
	Category         string
}

var categories = map[string]string{
	"people":             "Smileys & People",
	"animals_and_nature": "Animals & Nature",
	"food_and_drink":     "Food & Drink",
	"activity":           "Activity",
	"travel_and_places":  "Travel & Places",
	"objects":            "Objects",
	"symbols":            "Symbols",
	"flags":              "Flags",
}

// https://en.wikipedia.org/wiki/Fitzpatrick_scale
var fitzpatrickScaleModifiers = map[string]string{
	"skin_tone_1": "🏻",
	"skin_tone_2": "🏼",
	"skin_tone_3": "🏽",
	"skin_tone_4": "🏾",
	"skin_tone_5": "🏿",
}

// skin tone options
var skinToneSelections = []string{"✋", "✋🏻", "✋🏼", "✋🏽", "✋🏾", "✋🏿"}

func loadUniversalEmojis() map[string]emoji {
	// try to read emoji.json
	lib, err := ioutil.ReadFile("./db/emojis.json")

	if err != nil {
		if os.IsNotExist(err) {
			// doesn't exist, grab from emojilib and create emojis.json
			fmt.Println("No emojis.json found. Fetching..")
			lib = fetchUniversalEmojis()
			ioutil.WriteFile("emojis.json", lib, 0777)
			err := os.Rename("./emojis.json", "./db/emojis.json")
			if err != nil {
				panic(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	var universalEmojis map[string]emoji
	json.Unmarshal([]byte(string(lib)), &universalEmojis)
	return universalEmojis
}

func fetchUniversalEmojis() []byte {
	resp, err := http.Get("https://raw.githubusercontent.com/muan/emojilib/master/emojis.json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func addModifier(e emoji, modifier string) string {
	// no modifier or can't be modified
	if modifier == "" || !e.FitzpatrickScale {
		return e.Char
	}

	// skin tone magic explained: https://emojipedia.org/zero-width-joiner/
	// zwj := regexp.MustCompile("‍")
	// matches := zwj.FindAllString(emoji["char"])

	zwj := "‍"
	match, _ := regexp.Match(zwj, []byte(e.Char))

	if match {
		return strings.Replace(e.Char, zwj, modifier+zwj, -1)
	}
	return e.Char + modifier

}

func mergeMaps(o map[string]emoji, st map[string]emoji) map[string]emoji {
	merger := make(map[string]emoji, len(st))

	for name, emoji := range o {
		merger[name] = emoji
	}

	// skin toned emoji map
	for name, emoji := range st {
		if _, ok := merger[name]; ok {
			merger[name] = emoji
		}
	}

	return merger
}

func fetchSkinTonesHandler(w http.ResponseWriter, r *http.Request) {
	// load emojis to replace skin-toneable emojis with preferred skin tone emojis
	emojis := loadUniversalEmojis()

	// get modifier value
	tone := r.URL.Query()["skintone"][0]
	skinToneModifier := fitzpatrickScaleModifiers[tone]

	// grab emojis that can have their skin tone changed (e.g. "fitzpatrick_scale": true)
	skinTonableEmojis := make(map[string]emoji, len(emojis))
	for name, emoji := range emojis {
		if emoji.FitzpatrickScale {
			skinTonableEmojis[name] = emoji
		}
	}

	// change skin tones
	skinTonedEmojis := make(map[string]emoji, len(skinTonableEmojis))
	for name, emoji := range skinTonableEmojis {
		emoji.Char = addModifier(emoji, skinToneModifier)
		skinTonedEmojis[name] = emoji
	}

	// merge skin-toneable emojis back into emojis map
	updatedEmojis := mergeMaps(emojis, skinTonedEmojis)

	// update cookie with skin tone preference
	expiration := time.Now().Add(365 * 24 * time.Hour)
	cookie := http.Cookie{
		Name:    "tone",
		Value:   string(tone[len(tone)-1]),
		Expires: expiration,
	}
	http.SetCookie(w, &cookie)

	// send to template
	data := struct {
		Emojis             map[string]emoji
		Categories         map[string]string
		SkinToneSelections []string
		Hand               string
	}{
		updatedEmojis,
		categories,
		skinToneSelections,
		"",
	}

	renderTemplate(w, "emojis", data)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// get emojis
	emojis := loadUniversalEmojis()

	// skin tone preference
	var hand string
	c, err := r.Cookie("tone")
	if err != nil {
		// existing preference not set, use default
		hand = skinToneSelections[0]
	} else {
		preference, _ := strconv.Atoi(c.Value)
		hand = skinToneSelections[preference]
	}

	// send to template
	data := struct {
		Emojis             map[string]emoji
		Categories         map[string]string
		SkinToneSelections []string
		Hand               string
	}{
		emojis,
		categories,
		skinToneSelections,
		hand,
	}

	renderTemplate(w, "index", data)
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	// write content type to head
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// render layout
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	// serve stuff from static
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/fetch-skin-tones", fetchSkinTonesHandler)

	// listen on port
	log.Println("Listening on :4567...")
	log.Fatal(http.ListenAndServe(":4567", nil))
}
