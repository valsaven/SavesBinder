package main

import (
	"strings"

	"github.com/jeandeaual/go-locale"
)

// systemLanguageCode returns a BCP 47-ish language code from the OS (e.g. "ru", "en-US").
func systemLanguageCode() (string, error) {
	lang, err := locale.GetLanguage()
	if err == nil && lang != "" {
		return lang, nil
	}

	// Some platforms only expose full locale tags
	full, err2 := locale.GetLocale()
	if err2 != nil {
		if err != nil {
			return "", err
		}
		return "", err2
	}
	full = strings.ReplaceAll(full, "_", "-")
	if i := strings.IndexByte(full, '-'); i > 0 {
		return full[:i], nil
	}
	return full, nil
}
