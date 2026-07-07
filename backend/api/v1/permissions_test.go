package apiv1

import "testing"

func TestCanViewJobs(t *testing.T) {
	cases := []struct {
		name string
		perm *Permissions
		want bool
	}{
		{"empty / permission-less", &Permissions{}, false},
		{"admin", &Permissions{Admin: true}, true},
		{"export destination", &Permissions{Export: &ExportPermission{Destinations: []string{"vod"}}}, true},
		{"export timed metadata", &Permissions{Export: &ExportPermission{TimedMetadata: true}}, true},
		{"vb export destination", &Permissions{VbExport: &VBExportPermission{Destinations: []string{"gfx"}}}, true},
		{"bmm languages", &Permissions{Bmm: &BMMPermission{Languages: []string{"nor"}}}, true},
		{"bmm podcasts", &Permissions{Bmm: &BMMPermission{Podcasts: []string{"p1"}}}, true},
		{"transcription mediabanken", &Permissions{Transcription: &TranscriptionPermission{Mediabanken: true}}, true},
		{"vault enabled", &Permissions{Vault: &VaultPermission{Enabled: true}}, true},
		{"empty sub-permissions", &Permissions{
			Export:        &ExportPermission{},
			Bmm:           &BMMPermission{},
			Transcription: &TranscriptionPermission{},
			Vault:         &VaultPermission{},
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.perm.CanViewJobs(); got != c.want {
				t.Errorf("CanViewJobs() = %v, want %v", got, c.want)
			}
		})
	}
}
