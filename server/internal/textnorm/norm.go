package textnorm

import "strings"

func Normalize(s string) string {
	// Arabic yeh/kaf to Persian, strip diacritics and ZWNJ? keep simple
	s = strings.ReplaceAll(s, "ي", "ی")
	s = strings.ReplaceAll(s, "ك", "ک")
	// persian/arabic digits -> ascii
	for i := 0; i <= 9; i++ {
		ar := rune('٠' + i)
		fa := rune('۰' + i)
		s = strings.ReplaceAll(s, string(ar), string(rune('0'+i)))
		s = strings.ReplaceAll(s, string(fa), string(rune('0'+i)))
	}
	// remove ZWNJ
	s = strings.ReplaceAll(s, "\u200c", "")
	return s
}

func Equal(a, b string) bool { return Normalize(a) == Normalize(b) }
