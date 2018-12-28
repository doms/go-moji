package emoji

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

	"github.com/doms/go-moji/utils"
)

var (
	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"wrap":   utils.Wrap,
		"concat": utils.Concat,
	}).ParseGlob("templates/*"))

	skinToneSelections = []string{"👋", "👋🏻", "👋🏼", "👋🏽", "👋🏾", "👋🏿"}

	emojis            = loadEmojis()
	orderedEmojiNames = loadOrderedEmojis()

	// skin tone selector
	hand string

	categories = map[string]string{
		"people":             "Smileys & People",
		"animals_and_nature": "Animals & Nature",
		"food_and_drink":     "Food & Drink",
		"activity":           "Activity",
		"travel_and_places":  "Travel & Places",
		"objects":            "Objects",
		"symbols":            "Symbols",
		"flags":              "Flags",
	}

	orderedCategoryNames = []string{
		"people",
		"animals_and_nature",
		"food_and_drink",
		"activity",
		"travel_and_places",
		"objects",
		"symbols",
		"flags",
	}

	// https://en.wikipedia.org/wiki/Fitzpatrick_scale
	fitzpatrickScaleModifiers = map[string]string{
		"skin_tone_1": "🏻",
		"skin_tone_2": "🏼",
		"skin_tone_3": "🏽",
		"skin_tone_4": "🏾",
		"skin_tone_5": "🏿",
	}
)

type emoji struct {
	Keywords         []string `json:"keywords"`
	Char             string   `json:"char"`
	FitzpatrickScale bool     `json:"fitzpatrick_scale"`
	Category         string   `json:"category"`
}

func loadEmojis() map[string]emoji {
	// try to read emoji.json
	pwd, _ := os.Getwd()
	lib, err := ioutil.ReadFile(pwd + "/db/emojis.json")
	if err != nil {
		if os.IsNotExist(err) {
			// doesn't exist, grab from emojilib and create emojis.json
			fmt.Println("No emojis.json found. Fetching..")
			lib = fetchEmojis()
			ioutil.WriteFile("emojis.json", lib, 0777)
			err := os.Rename("./emojis.json", pwd+"/db/emojis.json")
			if err != nil {
				panic(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	var emojis map[string]emoji
	json.Unmarshal(lib, &emojis)
	return emojis
}

func loadOrderedEmojis() []string {
	// try to read ordered.json
	pwd, _ := os.Getwd()
	lib, err := ioutil.ReadFile(pwd + "/db/ordered.json")
	if err != nil {
		if os.IsNotExist(err) {
			// doesn't exist, grab from emojilib and create emojis.json
			fmt.Println("No ordered.json found. Fetching..")
			lib = fetchOrderedEmojis()
			ioutil.WriteFile("ordered.json", lib, 0777)
			err := os.Rename("./ordered.json", pwd+"/db/ordered.json")
			if err != nil {
				panic(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	var ordered []string
	json.Unmarshal(lib, &ordered)
	return ordered
}

func fetchEmojis() []byte {
	resp, err := http.Get("https://raw.githubusercontent.com/muan/emojilib/master/emojis.json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

func fetchOrderedEmojis() []byte {
	resp, err := http.Get("https://raw.githubusercontent.com/muan/emojilib/master/ordered.json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}

// addModifier - it's what makes skintones possible
func addModifier(e emoji, modifier string) string {
	// no modifier or can't be modified
	if modifier == "" || !e.FitzpatrickScale {
		return e.Char
	}

	// skin tone magic explained: https://emojipedia.org/zero-width-joiner/
	zwj := "‍"
	match, _ := regexp.Match(zwj, []byte(e.Char))

	if match {
		return strings.Replace(e.Char, zwj, modifier+zwj, -1)
	}

	return e.Char + modifier
}

// mergeMaps - merge maps of non-skintone-able emojis and skintone-able emojis
func mergeMaps(o map[string]emoji, st map[string]emoji) map[string]emoji {
	merger := make(map[string]emoji, len(st))

	for name, emoji := range o {
		merger[name] = emoji
	}

	for name, emoji := range st {
		if _, ok := merger[name]; ok {
			merger[name] = emoji
		}
	}

	return merger
}

// IndexHandler - renders the emojis (with skin tone preference if applicable)
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if emojis == nil {
		emojis = loadEmojis()
	}

	// skin tone preference
	c, err := r.Cookie("tone")
	if err != nil {
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
		OrderedCategories  []string
		OrderedEmojis      []string
		Hand               string
	}{
		emojis,
		categories,
		skinToneSelections,
		orderedCategoryNames,
		orderedEmojiNames,
		hand,
	}

	renderTemplate(w, "index", data)
}

// FetchSkinTonesHandler - add skin tones to skintone-able emojis
func FetchSkinTonesHandler(w http.ResponseWriter, r *http.Request) {
	if emojis == nil {
		emojis = loadEmojis()
	}

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
		OrderedCategories  []string
		OrderedEmojis      []string
		Hand               string
	}{
		updatedEmojis,
		categories,
		skinToneSelections,
		orderedCategoryNames,
		orderedEmojiNames,
		hand,
	}

	renderTemplate(w, "categories", data)
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
