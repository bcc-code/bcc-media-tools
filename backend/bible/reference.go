// Package bible resolves the free-text bible references editors type into
// verse text from the BCC bible server (https://bcc-code.gitbook.io/bible-server).
package bible

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ErrUnparsable is returned for a reference this package cannot make sense of —
// a misspelled book, a missing chapter, free text that is not a reference.
var ErrUnparsable = errors.New("bible: not a verse reference")

// VerseRange is a run of verses within a chapter. To == From for a single verse.
type VerseRange struct {
	From int
	To   int
}

// Reference is a parsed reference: one chapter of one book, and the verse runs
// picked out of it. Editors write these Norwegian-style — "1Kor 1,18+23-24" is
// chapter 1, verse 18 plus verses 23 through 24.
type Reference struct {
	// Book is the canonical short ID the bible server takes, e.g. "1Cor".
	Book string
	// BookName is the Norwegian name for display, e.g. "1. Korinterbrev".
	BookName string
	Chapter  int
	// Ranges is empty when the reference names a whole chapter.
	Ranges []VerseRange
}

// splitRef pulls the book token off the front. Books may start with a digit and
// carry a dot and a space ("1Kor", "1. Mos"), so the chapter is the first
// number that follows the book's letters.
var splitRef = regexp.MustCompile(`^\s*((?:[0-9]\s*\.?\s*)?[^0-9,:]+?)\s*([0-9].*)$`)

// verseRun matches one verse or verse range within the verse part.
var verseRun = regexp.MustCompile(`([0-9]+)\s*(?:[-–—]\s*([0-9]+))?`)

// Parse reads one reference. It accepts the Norwegian notation used in
// Mediabanken (comma between chapter and verse, "+" or "." between verse runs,
// hyphen or dash for a range) as well as the colon form, and both Norwegian and
// canonical book abbreviations.
func Parse(s string) (Reference, error) {
	m := splitRef.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return Reference{}, fmt.Errorf("%w: %q", ErrUnparsable, s)
	}

	b, ok := lookupBook(m[1])
	if !ok {
		return Reference{}, fmt.Errorf("%w: unknown book %q", ErrUnparsable, strings.TrimSpace(m[1]))
	}

	rest := strings.TrimSpace(m[2])
	chapterPart, versePart, hasVerses := strings.Cut(rest, ",")
	if !hasVerses {
		chapterPart, versePart, hasVerses = strings.Cut(rest, ":")
	}

	chapter, err := strconv.Atoi(strings.TrimSpace(chapterPart))
	if err != nil || chapter < 1 {
		return Reference{}, fmt.Errorf("%w: no chapter in %q", ErrUnparsable, s)
	}

	ref := Reference{Book: b.ID, BookName: b.Name, Chapter: chapter}
	if !hasVerses {
		return ref, nil
	}

	for _, run := range verseRun.FindAllStringSubmatch(versePart, -1) {
		from, err := strconv.Atoi(run[1])
		if err != nil || from < 1 {
			continue
		}
		to := from
		if run[2] != "" {
			if parsed, err := strconv.Atoi(run[2]); err == nil && parsed >= from {
				to = parsed
			}
		}
		ref.Ranges = append(ref.Ranges, VerseRange{From: from, To: to})
	}
	if len(ref.Ranges) == 0 {
		return Reference{}, fmt.Errorf("%w: no verses in %q", ErrUnparsable, s)
	}
	return ref, nil
}

// Display renders the reference the way it reads in Norwegian, e.g.
// "1. Korinterbrev 1,18+23–24".
func (r Reference) Display() string {
	var sb strings.Builder
	sb.WriteString(r.BookName)
	sb.WriteString(" ")
	sb.WriteString(strconv.Itoa(r.Chapter))
	for i, run := range r.Ranges {
		if i == 0 {
			sb.WriteString(",")
		} else {
			sb.WriteString("+")
		}
		sb.WriteString(strconv.Itoa(run.From))
		if run.To != run.From {
			sb.WriteString("–")
			sb.WriteString(strconv.Itoa(run.To))
		}
	}
	return sb.String()
}
