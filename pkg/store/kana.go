package store

import "slices"

// kanaFoldedLanguages name the fields where hiragana and katakana match each other.
// Only Japanese writes one word in either script; chs, tc and ko carry kana only as
// untranslated Japanese, and folding there would merely widen the noise.
var kanaFoldedLanguages = []string{"ja"}

func foldsKana(lang string) bool {
	return slices.Contains(kanaFoldedLanguages, lang)
}

// kanaVariantBoost weights a query term matched only in the script the user did not
// type. It has to be low enough that the typed script still wins and high enough that a
// name typed in the wrong script still beats rows matching a bigram or two as typed.
//
// Measured on publish-20260728-3b64471. By hand, with the ja field alone: at 1.0 a
// hiragana greeting got its katakana spelling mixed into the top ranks, and at 0.3 a
// long katakana name typed in hiragana lost the first page to rows sharing part of it.
//
// The relevance eval then compared 0.3, 0.5 and 0.7. All three find every case typed in
// the other script within ten (recall@10 1.000 for both kana-script families, from 0.194
// and 0.667 before) and pass every other family. By MRR on those families 0.3 trails
// (0.862 / 0.727) and 0.5 and 0.7 nearly tie (0.972 / 0.917 against 0.986 / 0.917), but
// 0.7 lets a row matching partly in the rarer katakana outrank the typed hiragana rows
// and moves four other cases down a rank for the two it lifts, so 0.5 stays.
const kanaVariantBoost = 0.5

// The hiragana block U+3041–U+3096 and the katakana block U+30A1–U+30F6 line up one to
// one, as do the iteration marks ゝゞ and ヽヾ.
const (
	hiraganaFirst  = 0x3041
	hiraganaLast   = 0x3096
	hiraIterFirst  = 0x309D
	hiraIterLast   = 0x309E
	katakanaOffset = 0x60
)

// kanaCounterpart returns the same kana in the other script. ー and ・ are shared by
// both scripts, and ヷヸヹヺ, ヿ, ゟ and the small katakana extensions have no single
// code point counterpart, so those report false like every non-kana rune.
func kanaCounterpart(r rune) (rune, bool) {
	switch {
	case r >= hiraganaFirst && r <= hiraganaLast,
		r >= hiraIterFirst && r <= hiraIterLast:
		return r + katakanaOffset, true
	case r >= hiraganaFirst+katakanaOffset && r <= hiraganaLast+katakanaOffset,
		r >= hiraIterFirst+katakanaOffset && r <= hiraIterLast+katakanaOffset:
		return r - katakanaOffset, true
	}
	return 0, false
}

// kanaVariants returns term followed by every other spelling reached by switching any
// of its kana to the other script. The cjk analyzer emits bigrams, so a term carries at
// most two kana and at most four spellings.
func kanaVariants(term string) []string {
	variants := []string{""}
	for _, r := range term {
		other, ok := kanaCounterpart(r)
		next := make([]string, 0, 2*len(variants))
		for _, prefix := range variants {
			next = append(next, prefix+string(r))
			if ok {
				next = append(next, prefix+string(other))
			}
		}
		variants = next
	}
	return variants
}

// foldKana turns every katakana with a counterpart into hiragana, so two spellings of
// one word compare equal.
func foldKana(s string) string {
	runes := []rune(s)
	for i, r := range runes {
		if r >= hiraganaFirst+katakanaOffset {
			if other, ok := kanaCounterpart(r); ok {
				runes[i] = other
			}
		}
	}
	return string(runes)
}
