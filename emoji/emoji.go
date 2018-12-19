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
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	// https://stackoverflow.com/a/38644571
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)

	// allowed templates
	// https://stackoverflow.com/a/46201881
	templates = template.Must(template.New("").Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values) == 0 {
				return nil, errors.New("invalid dict call")
			}

			dict := make(map[string]interface{})

			for i := 0; i < len(values); i++ {
				key, isset := values[i].(string)
				if !isset {
					if reflect.TypeOf(values[i]).Kind() == reflect.Map {
						m := values[i].(map[string]interface{})
						for i, v := range m {
							dict[i] = v
						}
					} else {
						return nil, errors.New("dict values must be maps")
					}
				} else {
					i++
					if i == len(values) {
						return nil, errors.New("specify the key for non array values")
					}
					dict[key] = values[i]
				}

			}
			return dict, nil
		},
	}).ParseGlob("templates/*"))

	// skin tones
	skinToneSelections = []string{"✋", "✋🏻", "✋🏼", "✋🏽", "✋🏾", "✋🏿"}

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
	Keywords         []string
	Char             string
	FitzpatrickScale bool
	Category         string
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

	var Emojis map[string]emoji
	json.Unmarshal([]byte(string(lib)), &Emojis)
	return Emojis
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
	emojis := loadEmojis()

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

// FetchSkinTonesHandler - add skin tones to skintone-able emojis
func FetchSkinTonesHandler(w http.ResponseWriter, r *http.Request) {
	// load emojis to replace skin-toneable emojis with preferred skin tone emojis
	emojis := loadEmojis()

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

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	// write content type to head
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// render layout
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
