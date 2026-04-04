// Package resolver implements the logic to resolve domain lists with inclusions, handling cycles and providing detailed error context for debugging issues in the source lists. The main function is Resolve, which takes a map of parsed lists and returns a map of list names to their fully resolved entries.
package resolver

import (
	"context"
	"fmt"
	"slices"

	"github.com/v2fly/domain-list-community/v2/logger"
	"github.com/v2fly/domain-list-community/v2/model"
)

// package level variables to hold the state during resolution
// Note: We avoid package level variables for state to make it thread-safe / reusable.

type resolverState struct {
	RawMap    map[string]model.ParsedList
	Resolving map[string]bool
	Resolved  map[string][]model.Entry
}

// Resolve takes a map of parsed lists and resolves all inclusions, returning a map of list names to their fully resolved entries. The running mechanism is a depth-first search with cycle detection, and includes detailed error context for easier debugging of issues in the source lists.
func Resolve(ctx context.Context, parsedListMap map[string]model.ParsedList) (map[string][]model.Entry, error) {
	state := &resolverState{
		RawMap:    parsedListMap,
		Resolving: make(map[string]bool),
		Resolved:  make(map[string][]model.Entry),
	}

	for name := range state.RawMap {
		if err := resolveOne(ctx, name, state); err != nil {
			logger.ErrorContext(ctx, "failed to resolve list %q: %v", name, err)
			return nil, fmt.Errorf("failed to resolve list %q: %w", name, err)
		}
	}

	return state.Resolved, nil
}

func resolveOne(ctx context.Context, name string, state *resolverState) error {
	// dfs baseline: if already resolved, return; if currently resolving, circular inclusion error
	if _, done := state.Resolved[name]; done {
		return nil
	}
	if state.Resolving[name] {
		logger.ErrorContext(ctx, "circular inclusion detected: %s", name)
		return fmt.Errorf("circular inclusion: %s", name)
	}

	state.Resolving[name] = true
	defer delete(state.Resolving, name)

	parsedList, exists := state.RawMap[name]
	if !exists {
		logger.ErrorContext(ctx, "list %q not globally defined", name)
		return fmt.Errorf("list %q not globally defined", name)
	}

	uniqueEntries := make(map[string]model.Entry)
	for _, entry := range parsedList.Entries {
		uniqueEntries[entry.Hash()] = entry
	}

	for _, inc := range parsedList.Inclusions {
		// dfs recursive resolution of included list, with error context for better debugging
		if err := resolveOne(ctx, inc.Target, state); err != nil {
			logger.ErrorContext(ctx, "failed to resolve included list %q in %q: %v", inc.Target, name, err)
			return fmt.Errorf("nested include %s inside %s: %w", inc.Target, name, err)
		}
		// filter included entries based on inclusion rules, and add matching entries to the uniqueEntries map
		for _, subEntry := range state.Resolved[inc.Target] {
			if matchInclude(&subEntry, &inc) {
				uniqueEntries[subEntry.Hash()] = subEntry
			}
		}
	}

	// convert uniqueEntries map back to a slice and store in resolved map
	resolvedEntries := make([]model.Entry, 0, len(uniqueEntries))
	for _, entry := range uniqueEntries {
		resolvedEntries = append(resolvedEntries, entry)
	}
	state.Resolved[name] = resolvedEntries
	return nil
}

func matchInclude(entry *model.Entry, inc *model.Inclusion) bool {
	if len(entry.Attrs) == 0 {
		return len(inc.MustAttrs) == 0
	}
	// Check that all required attributes are present and no banned attributes are present
	for _, req := range inc.MustAttrs {
		if !slices.Contains(entry.Attrs, req) {
			return false
		}
	}
	// Check that no banned attributes are present
	for _, ban := range inc.BanAttrs {
		if slices.Contains(entry.Attrs, ban) {
			return false
		}
	}
	return true
}
