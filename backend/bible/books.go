package bible

import "strings"

// book is one bible book: the canonical short ID the bible server takes (see
// the "Canonical representation of Verses" page of
// https://bcc-code.gitbook.io/bible-server), a Norwegian display name, and the
// spellings a reference may use.
type book struct {
	ID string
	// Ordinal is 1/2/3 for numbered books ("2. Krønikebok"), 0 otherwise. It
	// is matched exactly, so "2Kr" can never resolve to an unnumbered book.
	Ordinal int
	Name    string
	// Forms are the book's names without the ordinal, normalised. They are
	// prefix-matched, so listing "krønikebok" also covers "Krøn", "Krø", "Kr".
	Forms []string
}

// books is the lookup table. Norwegian names come first because that is what
// editors type; the English name and the canonical ID are there because
// references are hand-written and mix languages.
var books = []book{
	{"Gen", 1, "1. Mosebok", []string{"mosebok", "mos", "genesis", "gen"}},
	{"Exo", 2, "2. Mosebok", []string{"mosebok", "mos", "exodus", "exo"}},
	{"Lev", 3, "3. Mosebok", []string{"mosebok", "mos", "levitikus", "leviticus", "lev"}},
	{"Num", 4, "4. Mosebok", []string{"mosebok", "mos", "numeri", "numbers", "num"}},
	{"Deu", 5, "5. Mosebok", []string{"mosebok", "mos", "deuteronomium", "deuteronomy", "deu"}},
	{"Josh", 0, "Josva", []string{"josva", "joshua"}},
	{"Judg", 0, "Dommerne", []string{"dommerne", "dommernes", "judges"}},
	{"Ruth", 0, "Rut", []string{"rut", "ruth"}},
	{"1Sam", 1, "1. Samuelsbok", []string{"samuelsbok", "samuel", "sam"}},
	{"2Sam", 2, "2. Samuelsbok", []string{"samuelsbok", "samuel", "sam"}},
	{"1Kgs", 1, "1. Kongebok", []string{"kongebok", "kongebok", "konge", "kings", "kgs"}},
	{"2Kgs", 2, "2. Kongebok", []string{"kongebok", "kongebok", "konge", "kings", "kgs"}},
	{"1Chr", 1, "1. Krønikebok", []string{"krønikebok", "krøniker", "krønikerne", "chronicles", "chr"}},
	{"2Chr", 2, "2. Krønikebok", []string{"krønikebok", "krøniker", "krønikerne", "chronicles", "chr"}},
	{"Ezra", 0, "Esra", []string{"esra", "ezra"}},
	{"Neh", 0, "Nehemja", []string{"nehemja", "nehemiah"}},
	{"Est", 0, "Ester", []string{"ester", "esther"}},
	{"Job", 0, "Job", []string{"job"}},
	{"Ps", 0, "Salmene", []string{"salmene", "salme", "salmenes", "psalms", "psalm"}},
	{"Prov", 0, "Ordspråkene", []string{"ordspråkene", "ordspråk", "proverbs", "prov"}},
	{"Eccl", 0, "Forkynneren", []string{"forkynneren", "predikeren", "ecclesiastes", "eccl"}},
	{"Song", 0, "Høysangen", []string{"høysangen", "høysang", "songofsongs", "song"}},
	{"Isa", 0, "Jesaja", []string{"jesaja", "isaiah", "isa"}},
	{"Jer", 0, "Jeremia", []string{"jeremia", "jeremiah"}},
	{"Lam", 0, "Klagesangene", []string{"klagesangene", "klagesang", "lamentations", "lam"}},
	{"Ezek", 0, "Esekiel", []string{"esekiel", "ezekiel"}},
	{"Dan", 0, "Daniel", []string{"daniel"}},
	{"Hos", 0, "Hosea", []string{"hosea"}},
	{"Joel", 0, "Joel", []string{"joel"}},
	{"Am", 0, "Amos", []string{"amos"}},
	{"Ob", 0, "Obadja", []string{"obadja", "obadiah"}},
	{"Jon", 0, "Jona", []string{"jona", "jonas", "jonah"}},
	{"Mic", 0, "Mika", []string{"mika", "micah"}},
	{"Nah", 0, "Nahum", []string{"nahum"}},
	{"Hab", 0, "Habakkuk", []string{"habakkuk"}},
	{"Zeph", 0, "Sefanja", []string{"sefanja", "zephaniah"}},
	{"Hag", 0, "Haggai", []string{"haggai"}},
	{"Zech", 0, "Sakarja", []string{"sakarja", "zechariah"}},
	{"Mal", 0, "Malaki", []string{"malaki", "malachi"}},
	{"Mt", 0, "Matteus", []string{"matteus", "matthew", "mt"}},
	{"Mk", 0, "Markus", []string{"markus", "mark", "mk"}},
	{"Lk", 0, "Lukas", []string{"lukas", "luke", "lk"}},
	{"Jn", 0, "Johannes", []string{"johannes", "john", "jn"}},
	{"Acts", 0, "Apostlenes gjerninger", []string{"apostlenesgjerninger", "apg", "acts"}},
	{"Rom", 0, "Romerne", []string{"romerne", "romerbrevet", "romans"}},
	{"1Cor", 1, "1. Korinterbrev", []string{"korinterbrev", "korinterne", "korinter", "corinthians", "cor"}},
	{"2Cor", 2, "2. Korinterbrev", []string{"korinterbrev", "korinterne", "korinter", "corinthians", "cor"}},
	{"Gal", 0, "Galaterne", []string{"galaterne", "galaterbrevet", "galatians"}},
	{"Eph", 0, "Efeserne", []string{"efeserne", "efeserbrevet", "ephesians", "eph"}},
	{"Phil", 0, "Filipperne", []string{"filipperne", "filipperbrevet", "philippians"}},
	{"Col", 0, "Kolosserne", []string{"kolosserne", "kolosserbrevet", "colossians", "col"}},
	{"1Ths", 1, "1. Tessalonikerbrev", []string{"tessalonikerbrev", "tessalonikerne", "tessaloniker", "thessalonians", "ths"}},
	{"2Ths", 2, "2. Tessalonikerbrev", []string{"tessalonikerbrev", "tessalonikerne", "tessaloniker", "thessalonians", "ths"}},
	{"1Tim", 1, "1. Timoteus", []string{"timoteus", "timothy", "tim"}},
	{"2Tim", 2, "2. Timoteus", []string{"timoteus", "timothy", "tim"}},
	{"Titus", 0, "Titus", []string{"titus"}},
	{"Phlm", 0, "Filemon", []string{"filemon", "philemon", "phlm"}},
	{"Heb", 0, "Hebreerne", []string{"hebreerne", "hebreerbrevet", "hebrews", "heb"}},
	{"Jas", 0, "Jakob", []string{"jakob", "jakobs", "james", "jas"}},
	{"1Pet", 1, "1. Peter", []string{"peter", "peters", "pet"}},
	{"2Pet", 2, "2. Peter", []string{"peter", "peters", "pet"}},
	{"1Jn", 1, "1. Johannes", []string{"johannes", "john", "jn"}},
	{"2Jn", 2, "2. Johannes", []string{"johannes", "john", "jn"}},
	{"3Jn", 3, "3. Johannes", []string{"johannes", "john", "jn"}},
	{"Jude", 0, "Judas", []string{"judas", "jude"}},
	{"Rev", 0, "Åpenbaringen", []string{"åpenbaringen", "åpenbaring", "revelation", "rev"}},
}

// exactForms pins the standard abbreviations that prefix matching would call
// ambiguous: "Fil" is Filipperne (not Filemon), "Jud" is Judas (not Judges),
// "Åp" is Åpenbaringen (not Apostlenes gjerninger). Keyed on the normalised
// token with its accents intact, and checked before any prefix matching.
var exactForms = map[string]string{
	"fil":  "Phil",
	"jud":  "Jude",
	"judg": "Judg",
	"dom":  "Judg",
	"åp":   "Rev",
	"åpb":  "Rev",
	"ap":   "Rev",
	"apg":  "Acts",
	"sal":  "Ps",
	"ord":  "Prov",
}

var byID = func() map[string]book {
	m := make(map[string]book, len(books))
	for _, b := range books {
		m[b.ID] = b
	}
	return m
}()

// normalize lower-cases and drops what varies between spellings of the same
// book: dots, spaces and hyphens. Accents are kept — "Åp" must stay separable
// from "Ap(g)".
func normalize(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		switch r {
		case '.', ' ', '-', '\t', ' ':
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// fold flattens the Norwegian vowels so "Krø", "Kro" and "Kr" all reach
// "krønikebok".
func fold(s string) string {
	r := strings.NewReplacer("ø", "o", "æ", "a", "å", "a", "ö", "o", "ä", "a")
	return r.Replace(s)
}

// splitOrdinal separates a leading book number from the rest: "2Kr" → 2, "kr".
func splitOrdinal(s string) (int, string) {
	for i, r := range s {
		if r >= '1' && r <= '5' {
			continue
		}
		if i == 0 {
			return 0, s
		}
		return int(s[0] - '0'), s[i:]
	}
	return 0, s
}

// lookupBook resolves a book token. Exact standard abbreviations win; anything
// else is prefix-matched against the book's names, with the ordinal having to
// match, and is only accepted when exactly one book fits.
func lookupBook(token string) (book, bool) {
	norm := normalize(token)
	if id, ok := exactForms[norm]; ok {
		return byID[id], true
	}

	ordinal, letters := splitOrdinal(norm)
	if id, ok := exactForms[letters]; ok {
		// "1Joh" and "2Joh" reach their numbered books, not the pinned one.
		if b := byID[id]; b.Ordinal == ordinal {
			return b, true
		}
	}
	if len(letters) < 2 {
		return book{}, false
	}
	letters = fold(letters)

	var match book
	found := 0
	for _, b := range books {
		if b.Ordinal != ordinal {
			continue
		}
		for _, form := range b.Forms {
			if strings.HasPrefix(fold(form), letters) {
				if found == 0 || match.ID != b.ID {
					found++
					match = b
				}
				break
			}
		}
	}
	if found != 1 {
		return book{}, false
	}
	return match, true
}
