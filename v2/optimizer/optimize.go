package optimizer

import (
	"context"
	"slices"
	"strings"

	"github.com/v2fly/domain-list-community/v2/model"
)

// Optimize performs deduplication and redundancy removal on a resolved list of entries.
// Specifically, it drops subdomain rules if their parent domain is already included as a generic domain rule.
// Example 1: If `domain:example.com` exists, `domain:www.example.com` is redundant and gets removed.
// Example 2: If `domain:example.com` exists, `full:example.com` is redundant and gets removed.
// Note: Rules with specific attributes, keywords, or regexps are never optimized away to preserve their specific filtering logic.
func Optimize(ctx context.Context, entries []model.Entry) []model.Entry {
	domains := make(map[string]bool)
	var final, candidates []model.Entry

	// Phase 1: Segregate entries.
	// Only pure domain and full rules (without attributes) are candidates for redundancy checks.
	for _, e := range entries {
		if (e.Type == model.TypeDomain || e.Type == model.TypeFull) && len(e.Attrs) == 0 {
			if e.Type == model.TypeDomain {
				// Mark this as a covering parent domain
				domains[e.Value] = true
			}
			candidates = append(candidates, e)
		} else {
			// Rules with attributes, regex, or keywords bypass optimization and go straight to final
			final = append(final, e)
		}
	}

	// Phase 2: Filter redundancies.
	// Check each candidate to see if any of its parent domains exist in the `domains` map.
	for _, c := range candidates {
		redundant := false
		target := c.Value
		// If it's a full domain rule (e.g., `full:example.com`), we prepend a dot (`.example.com`).
		// This forces the upcoming `strings.Cut` to first check `example.com` itself against the `domains` map.
		// If it's `domain:example.com`, it skips checking itself and only checks parent boundaries (like `com`).
		if c.Type == model.TypeFull {
			target = "." + target
		}
		for {
			// Peel off the first subdomain part (e.g. "www.example.com" -> parent="example.com")
			_, parent, hasParent := strings.Cut(target, ".")
			if !hasParent {
				break
			}
			if domains[parent] {
				redundant = true
				break
			}
			target = parent
		}
		if !redundant {
			final = append(final, c)
		}
	}

	// Phase 3: Sort deterministic output strings
	slices.SortFunc(final, func(a, b model.Entry) int {
		return strings.Compare(a.Hash(), b.Hash())
	})

	return final
}
