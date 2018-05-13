package main

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// stemmer is a fate stemmer that normalizes content for differences in
// case, punctuation, and accents. It also applies the snowball English
// stemmer to any tokens.
type stemmer struct {
	tran transform.Transformer
	lang string
}

func newStemmer(lang string) stemmer {
	isRemovable := func(r rune) bool {
		return unicode.Is(unicode.Mn, r) || unicode.IsPunct(r)
	}

	return stemmer{
		tran: transform.Chain(norm.NFD, transform.RemoveFunc(isRemovable), norm.NFC),
		lang: lang,
	}
}

func (s stemmer) Stem(word string) string {
	str, _, _ := transform.String(s.tran, word)
	stemmed, err := snowball.Stem(strings.ToLower(str), s.lang, false)
	if err != nil {
		return word
	}
	return stemmed
}
