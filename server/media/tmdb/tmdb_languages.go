package tmdb

import (
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/sbondCo/Watcharr/cache"
)

// LanguageOption is a selectable metadata language (for the TMDB_LANG setting).
type LanguageOption struct {
	Code string `json:"code"` // TMDB "language" value, e.g. "de-DE"
	Name string `json:"name"` // Human label, e.g. "German (DE)"
}

// Languages returns the list of TMDB-supported translation languages
// (primary translations), with human-readable labels, for a dropdown.
func (t *TMDB) Languages() ([]LanguageOption, error) {
	cacheKey := cache.CreateCacheKey("Languages")
	cached := new([]LanguageOption)
	if cache.GetCache(ContentStore, cacheKey, &cached) {
		slog.Debug("Languages: Returning cache.")
		return *cached, nil
	}
	var prim []string
	if err := t.req("/configuration/primary_translations", map[string]string{}, &prim); err != nil {
		slog.Error("Languages: primary_translations request failed", "error", err)
		return nil, errors.New("request failed")
	}
	var langs []struct {
		Iso6391     string `json:"iso_639_1"`
		EnglishName string `json:"english_name"`
	}
	if err := t.req("/configuration/languages", map[string]string{}, &langs); err != nil {
		slog.Error("Languages: languages request failed", "error", err)
		return nil, errors.New("request failed")
	}
	nameByIso := make(map[string]string, len(langs))
	for _, l := range langs {
		nameByIso[l.Iso6391] = l.EnglishName
	}
	out := make([]LanguageOption, 0, len(prim))
	for _, code := range prim {
		lang, region := code, ""
		if i := strings.Index(code, "-"); i > 0 {
			lang, region = code[:i], code[i+1:]
		}
		name := nameByIso[lang]
		if name == "" {
			name = lang
		}
		if region != "" {
			name = name + " (" + region + ")"
		}
		out = append(out, LanguageOption{Code: code, Name: name})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	ContentStore.Set(cacheKey, &out, time.Hour*24)
	return out, nil
}
