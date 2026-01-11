package registry

import "strings"

// FuzzyMatch returns true if query fuzzy-matches target.
// Matching is case-insensitive and succeeds on substring match or if
// the query characters appear as a subsequence in the target.
func FuzzyMatch(target, query string) bool {
	if query == "" {
		return true
	}
	t := strings.ToLower(target)
	q := strings.ToLower(query)
	if strings.Contains(t, q) {
		return true
	}
	// subsequence match (rune-aware)
	qr := []rune(q)
	i := 0
	for _, ch := range t {
		if i < len(qr) && qr[i] == ch {
			i++
			if i >= len(qr) {
				return true
			}
		}
	}
	return false
}

// fuzzyMatchesCommandSet returns true if the command set matches the query
// by checking name, description, commands, and tags.
func fuzzyMatchesCommandSet(cs *CommandSet, query string) bool {
	if FuzzyMatch(cs.Name, query) {
		return true
	}
	if cs.Description.Valid && FuzzyMatch(cs.Description.String, query) {
		return true
	}
	for _, c := range cs.Commands {
		if FuzzyMatch(c.Command, query) {
			return true
		}
	}
	for _, tg := range cs.Tags {
		if FuzzyMatch(tg, query) {
			return true
		}
	}
	return false
}

// FuzzySearchCommandSets searches command sets by fuzzy-matching name, description,
// commands, and tags. It loads full command sets and applies fuzzy matching in Go.
func (r *Repository) FuzzySearchCommandSets(query string) ([]CommandSet, error) {
	sets, err := r.ListCommandSets()
	if err != nil {
		return nil, err
	}
	var out []CommandSet
	for _, s := range sets {
		cs, err := r.GetCommandSetByName(s.Name)
		if err != nil {
			return nil, err
		}
		if cs == nil {
			continue
		}
		if fuzzyMatchesCommandSet(cs, query) {
			out = append(out, *cs)
		}
	}
	return out, nil
}
