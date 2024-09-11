package apiv1

// Implement sort.Interface for LanguageList

func (l *LanguageList) Len() int {
	return len(l.Languages)
}

func (l *LanguageList) Less(i, j int) bool {
	if l.Languages[i].Code == "nb" {
		return true
	} else if l.Languages[j].Code == "nb" {
		return false
	}

	return l.Languages[i].Code < l.Languages[j].Code
}

func (l *LanguageList) Swap(i, j int) {
	l.Languages[i], l.Languages[j] = l.Languages[j], l.Languages[i]
}
