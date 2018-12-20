package emoji

import (
	"encoding/json"
	"errors"
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

var (
	// allowed templates
	// https://stackoverflow.com/a/46201881
	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"wrap": func(values ...interface{}) (map[string]interface{}, error) {
			data := make(map[string]interface{}, len(values)/2)

			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				data[key] = values[i+1]
			}

			return data, nil
		},
		"concat": func(str string, strs ...string) string {
			// this is really gross and I feel bad for it.
			// but Go templates won't let me do it in a normal way...

			// join together emoji keywords with emoji name
			f := str + strings.Join(strs, " ")

			// remove slice brackets in string
			f = strings.Replace(f, "[", " ", -1)
			f = strings.Replace(f, "]", " ", -1)

			return strings.Trim(f, " ")
		},
	}).ParseGlob("templates/*"))

	// skin tones
	skinToneSelections = []string{"👋", "👋🏻", "👋🏼", "👋🏽", "👋🏾", "👋🏿"}

	// fetch emojis
	emojis            = loadEmojis()
	orderedEmojiNames = loadOrderedEmojis()

	// skin tone selector
	hand string

	// emoji categories
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

	// https://stackoverflow.com/a/19127931
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

// loadEmojis - gets emojis from json, and stores into map of custom type
func loadEmojis() map[string]emoji {
	// try to read emoji.json
	lib, err := ioutil.ReadFile(basepath + "/../db/emojis.json")

	if err != nil {
		if os.IsNotExist(err) {
			// doesn't exist, grab from emojilib and create emojis.json
			fmt.Println("No emojis.json found. Fetching..")
			lib = fetchEmojis()
			ioutil.WriteFile("emojis.json", lib, 0777)
			err := os.Rename("./emojis.json", basepath+"/../db/emojis.json")
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
	lib, err := ioutil.ReadFile(basepath + "/../db/ordered.json")
	if err != nil {
		log.Fatal(err)
	}

	var ordered []string
	json.Unmarshal(lib, &ordered)
	return ordered
}

// fetchEmojis - fetches from source if not available locally
func fetchEmojis() []byte {
	resp, err := http.Get("https://raw.githubusercontent.com/muan/emojilib/master/emojis.json")
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
	// zwj := regexp.MustCompile("‍")
	// matches := zwj.FindAllString(emoji["char"])

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

	// skin toned emoji map
	for name, emoji := range st {
		if _, ok := merger[name]; ok {
			merger[name] = emoji
		}
	}

	return merger
}

// IndexHandler - renders the emojis (with skin tone preference if applicable)
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// get emojis
	if emojis == nil {
		emojis = loadEmojis()
	}

	// skin tone preference
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
	// load emojis to replace skin-toneable emojis with preferred skin tone emojis
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
