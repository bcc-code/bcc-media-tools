package editorial

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanContributors(t *testing.T) {
	got := CleanContributors([]Contributor{
		{Name: " Ola Nordmann ", PersonID: " 90001 "},
		{Name: " ", PersonID: "90002"},
		{Name: "Ola"},
	})
	assert.Equal(t, []Contributor{{"Ola Nordmann", "90001"}, {"Ola", ""}}, got)
	assert.Empty(t, CleanContributors(nil))
}

func TestMatchesWords(t *testing.T) {
	for _, tc := range []struct {
		name, query string
		want        bool
	}{
		{"Ola Nordmann", "ola", true},
		{"Ola Nordmann", "nor ola", true},
		{"Ola Nordmann", "la", false},
		{"Ola Nordmann", "ola x", false},
		{"Kari Nordmann-Berg", "berg", true},
		{"K. Nordmann", "k", true},
		{"Åse Ødegård", "åse øde", true},
		{"Åse Ødegård", "ase", false},
		{"Øystein", "o", false},
	} {
		assert.Equal(t, tc.want, matchesWords(tc.name, tc.query), "%q in %q", tc.query, tc.name)
	}
}
