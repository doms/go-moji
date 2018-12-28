package utils

import (
	"errors"
	"strings"
)

// Wrap serves for storing key, val pairs in templates
func Wrap(values ...interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{}, len(values)/2)

	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		data[key] = values[i+1]
	}

	return data, nil
}

// Concat joins strings together in templates
func Concat(str string, strs ...string) string {
	// this is really gross and I feel bad for it.
	// but Go templates won't let me do it in a normal way...

	// join together emoji keywords with emoji name
	f := str + strings.Join(strs, " ")

	// remove slice brackets in string
	f = strings.Replace(f, "[", " ", -1)
	f = strings.Replace(f, "]", " ", -1)

	return strings.Trim(f, " ")
}
