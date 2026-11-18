package openai_compatible

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
)

// Go matches struct fields case-insensitively and otherwise accepts last-key
// wins. Reject ambiguous responses before a later field can hide a tool request.
func validateResponseKeys(raw []byte) error {
	if !json.Valid(raw) {
		return errors.New("invalid JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value func() error
	value = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'):
			seen := map[string]bool{}
			for decoder.More() {
				token, err := decoder.Token()
				if err != nil {
					return err
				}
				key := strings.Map(func(r rune) rune {
					for {
						next := unicode.SimpleFold(r)
						if next <= r {
							return next
						}
						r = next
					}
				}, token.(string))
				if seen[key] {
					return errors.New("duplicate response field")
				}
				seen[key] = true
				if err := value(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case json.Delim('['):
			for decoder.More() {
				if err := value(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		}
		return nil
	}
	return value()
}
