package cardsdb

import (
	"strings"
)

type SearchCardsParams struct {
	SearchTerm           string
	Class                string
	Type                 string
	Subtype              string
	Talent               string
	Keyword              string
	Cost                 string
	Defense              string
	Pitch                string
	Power                string
	Health               string
	Intelligence         string
	Arcane               string
	ColorIdentity        string
	Artist               string
	SetCode              string
	Rarity               string
	Language             string
	BlitzLegal           *bool
	BlitzBanned          *bool
	BlitzSuspended       *bool
	BlitzLivingLegend    *bool
	CCLegal              *bool
	CCBanned             *bool
	CCSuspended          *bool
	CCLivingLegend       *bool
	CommonerLegal        *bool
	CommonerBanned       *bool
	CommonerSuspended    *bool
	UPFBanned            *bool
	LLBanned             *bool
	LLRestricted         *bool
	ProjectBlueLegal     *bool
	ProjectBlueBanned    *bool
	ProjectBlueSuspended *bool
	IsDoubleFaced        *bool
	PageSize             int
	NextToken            string
}

func tokenizeSearchQuery(input string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false
	for _, r := range input {
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && (r == ' ' || r == '\t' || r == '\n' || r == '\r') {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func ParseCardSearchQuery(query string) SearchCardsParams {
	input := strings.TrimSpace(query)
	out := SearchCardsParams{}
	if input == "" {
		return out
	}
	var residual []string
	for _, token := range tokenizeSearchQuery(input) {
		if parseComparatorToken(token, &out) {
			continue
		}
		key, value, ok := strings.Cut(token, ":")
		if !ok || strings.TrimSpace(key) == "" {
			residual = append(residual, token)
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = cleanQueryValue(value)
		if value == "" {
			continue
		}
		if applyLegalSearchToken(key, value, &out) {
			continue
		}
		if applySpecialSearchToken(key, value, &out) {
			continue
		}
		if applyStringSearchToken(key, value, &out) {
			continue
		}
		if applyNumericSearchToken(key, value, &out) {
			continue
		}
		residual = append(residual, token)
	}
	out.SearchTerm = normalizeSearchWhitespace(strings.Join(residual, " "))
	return out
}

func parseComparatorToken(token string, out *SearchCardsParams) bool {
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if idx := strings.Index(token, op); idx > 0 {
			key := strings.ToLower(strings.TrimSpace(token[:idx]))
			value := cleanQueryValue(token[idx:])
			return applyNumericSearchToken(key, value, out)
		}
	}
	return false
}

func applyStringSearchToken(key, value string, out *SearchCardsParams) bool {
	switch key {
	case "class", "c":
		out.Class = normalizeSearchWhitespace(value)
	case "type", "t":
		out.Type = normalizeSearchWhitespace(value)
	case "subtype", "st":
		out.Subtype = normalizeSearchWhitespace(value)
	case "talent", "tal":
		out.Talent = normalizeSearchWhitespace(value)
	case "keyword", "kw":
		out.Keyword = normalizeSearchWhitespace(value)
	case "artist", "art":
		out.Artist = normalizeSearchWhitespace(value)
	case "set", "setcode", "s":
		out.SetCode = normalizeSearchWhitespace(value)
	case "rarity", "r":
		out.Rarity = normalizeSearchWhitespace(value)
	case "lang", "language":
		out.Language = normalizeSearchWhitespace(value)
	default:
		return false
	}
	return true
}

func applyNumericSearchToken(key, value string, out *SearchCardsParams) bool {
	switch key {
	case "cost", "co":
		out.Cost = normalizeNumericSearchValue(value)
	case "defense", "def", "d", "block", "b":
		out.Defense = normalizeNumericSearchValue(value)
	case "pitch", "p":
		out.Pitch = normalizeNumericSearchValue(value)
	case "power", "pow", "pwr", "attack":
		out.Power = normalizeNumericSearchValue(value)
	case "health", "life", "li", "hp":
		out.Health = normalizeNumericSearchValue(value)
	case "intelligence", "intellect", "i":
		out.Intelligence = normalizeNumericSearchValue(value)
	case "arcane":
		out.Arcane = normalizeNumericSearchValue(value)
	default:
		return false
	}
	return true
}

func applySpecialSearchToken(key, value string, out *SearchCardsParams) bool {
	switch key {
	case "color", "colour":
		for _, part := range splitSelectorValues(value) {
			switch strings.ToLower(part) {
			case "red":
				out.ColorIdentity = "COLOR_IDENTITY_RED"
				return true
			case "yellow":
				out.ColorIdentity = "COLOR_IDENTITY_YELLOW"
				return true
			case "blue":
				out.ColorIdentity = "COLOR_IDENTITY_BLUE"
				return true
			case "none":
				out.ColorIdentity = "COLOR_IDENTITY_NONE"
				return true
			}
		}
		return true
	case "double", "doublefaced", "double-faced", "twosided", "two-sided":
		if b, ok := parseSearchBool(value); ok {
			out.IsDoubleFaced = &b
		}
		return true
	default:
		return false
	}
}

func applyLegalSearchToken(key, value string, out *SearchCardsParams) bool {
	action := strings.ReplaceAll(strings.ToLower(key), " ", "")
	switch action {
	case "ll":
		action = "banned"
	case "legal", "banned", "suspended", "restricted", "livinglegend":
	default:
		return false
	}
	format := strings.ReplaceAll(strings.ToLower(value), " ", "")
	set := func(dst **bool) {
		v := true
		*dst = &v
	}
	switch format {
	case "blitz":
		switch action {
		case "legal":
			set(&out.BlitzLegal)
		case "banned":
			set(&out.BlitzBanned)
		case "suspended":
			set(&out.BlitzSuspended)
		case "livinglegend":
			set(&out.BlitzLivingLegend)
		}
	case "cc", "classicconstructed":
		switch action {
		case "legal":
			set(&out.CCLegal)
		case "banned":
			set(&out.CCBanned)
		case "suspended":
			set(&out.CCSuspended)
		case "livinglegend":
			set(&out.CCLivingLegend)
		}
	case "commoner":
		switch action {
		case "legal":
			set(&out.CommonerLegal)
		case "banned":
			set(&out.CommonerBanned)
		case "suspended":
			set(&out.CommonerSuspended)
		}
	case "upf":
		if action == "banned" {
			set(&out.UPFBanned)
		}
	case "ll", "livinglegend":
		switch action {
		case "banned":
			set(&out.LLBanned)
		case "restricted":
			set(&out.LLRestricted)
		}
	case "projectblue":
		switch action {
		case "legal":
			set(&out.ProjectBlueLegal)
		case "banned":
			set(&out.ProjectBlueBanned)
		case "suspended":
			set(&out.ProjectBlueSuspended)
		}
	}
	return true
}

func cleanQueryValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return strings.TrimSpace(value)
}

func normalizeNumericSearchValue(value string) string {
	value = cleanQueryValue(value)
	for _, op := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(value, op) {
			return op + strings.TrimSpace(strings.TrimPrefix(value, op))
		}
	}
	return strings.TrimSpace(value)
}

func normalizeSearchWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func parseSearchBool(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	default:
		return false, false
	}
}

func splitSelectorValues(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '+'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
