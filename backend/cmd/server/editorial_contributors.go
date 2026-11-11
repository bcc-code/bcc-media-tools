package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"encoding/json"
	"slices"
	"strings"
)

// contributor is one contributor_list entry; PersonID is empty when typed in.
type contributor struct {
	Name     string `json:"name"`
	PersonID string `json:"personId,omitempty"`
}

// markerContributors falls back to the contributors string for older rows.
func markerContributors(m editorial.Marker) []contributor {
	var list []contributor
	if m.ContributorList != "" {
		_ = json.Unmarshal([]byte(m.ContributorList), &list)
	}
	if len(list) == 0 {
		return splitContributors(m.Contributors)
	}
	return list
}

// setContributors also sets the contributors string to the joined names.
func setContributors(m *editorial.Marker, list []contributor) {
	m.Contributors = strings.Join(contributorNames(list), ", ")
	if len(list) == 0 {
		m.ContributorList = "[]"
		return
	}
	b, _ := json.Marshal(list)
	m.ContributorList = string(b)
}

// contributorsFromRequest uses the list unless the string names others; then
// the string wins and keeps IDs already linked in the list or stored row.
func contributorsFromRequest(list []*apiv1.EditorialContributor, legacy string, stored []contributor) []contributor {
	listed := make([]contributor, 0, len(list))
	for _, c := range list {
		name := strings.TrimSpace(c.GetName())
		if name == "" {
			continue
		}
		listed = append(listed, contributor{Name: name, PersonID: strings.TrimSpace(c.GetPersonId())})
	}
	typed := splitContributors(legacy)
	if len(listed) > 0 && (len(typed) == 0 || slices.Equal(contributorNames(listed), contributorNames(typed))) {
		return listed
	}
	ids := map[string]string{}
	for _, c := range slices.Concat(stored, listed) {
		if c.PersonID != "" {
			ids[c.Name] = c.PersonID
		}
	}
	for i := range typed {
		typed[i].PersonID = ids[typed[i].Name]
	}
	return typed
}

func contributorNames(list []contributor) []string {
	names := make([]string, len(list))
	for i, c := range list {
		names[i] = c.Name
	}
	return names
}

func splitContributors(s string) []contributor {
	var out []contributor
	for _, name := range strings.Split(s, ",") {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, contributor{Name: name})
		}
	}
	return out
}

// contributorListColumn writes "[]" for rows built without contributors.
func contributorListColumn(m editorial.Marker) string {
	if m.ContributorList == "" {
		return "[]"
	}
	return m.ContributorList
}

func contributorsChanged(a, b editorial.Marker) bool {
	return !slices.Equal(markerContributors(a), markerContributors(b))
}

func contributorsToProto(list []contributor) []*apiv1.EditorialContributor {
	out := make([]*apiv1.EditorialContributor, len(list))
	for i, c := range list {
		out[i] = &apiv1.EditorialContributor{Name: c.Name, PersonId: c.PersonID}
	}
	return out
}
