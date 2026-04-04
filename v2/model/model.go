// Package model defines the data structures used to represent parsed domain lists.
package model

// RuleType represents the type of a domain list entry
type RuleType string

const (
	TypeDomain  RuleType = "domain"
	TypeFull    RuleType = "full"
	TypeKeyword RuleType = "keyword"
	TypeRegexp  RuleType = "regexp"
	TypeInclude RuleType = "include"
)

// Entry represents a single entry in a domain list
type Entry struct {
	Type RuleType `json:"type"`
	// Value is the main content of the entry, such as a domain name or keyword
	Value string `json:"value"`
	// Attrs are optional attributes that modify the behavior of the entry
	Attrs []string `json:"attrs,omitempty"`
}

// Hash generates a unique string representation of the entry for comparison and deduplication purposes
func (e *Entry) Hash() string {
	hash := string(e.Type) + ":" + e.Value
	for _, a := range e.Attrs {
		hash += "@" + a
	}
	return hash
}

// Inclusion represents an inclusion rule in a domain list
type Inclusion struct {
	// Target is the name of the list to include
	Target string
	// MustAttrs are attributes that must be present in included entries for them to be included.
	// Marked as `@<attr>` in the source list, but stored as `<attr>` for easier processing.
	MustAttrs []string
	// BanAttrs are attributes that must not be present in included entries for them to be included.
	// Marked as `@-<attr>` in the source list, but stored as `<attr>` for easier processing.
	BanAttrs []string
}

// ParsedList represents a fully parsed domain list, including its entries and any inclusions
// Actually it's a single domain file under `data`, but "list" is more concise and less confusing than "file"
type ParsedList struct {
	Name       string
	Entries    []Entry
	Inclusions []Inclusion
}
