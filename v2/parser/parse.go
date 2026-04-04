// Package parser provides functions to parse domain lists in various formats
package parser

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/v2fly/domain-list-community/v2/logger"
	"github.com/v2fly/domain-list-community/v2/model"
)

func ParseDirectory(ctx context.Context, dir string) (map[string]model.ParsedList, error) {
	parsedListMap := make(map[string]model.ParsedList)
	if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		// Skip directories and hidden files
		if err != nil || info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			logger.ErrorContext(ctx, "error accessing path %q: %v", path, err)
			return err
		}

		// Parse the file and get the ParsedList and any affiliations
		parsedList, affMap, err := ParseFile(ctx, path)
		if err != nil {
			logger.ErrorContext(ctx, "failed to parse file %q: %v", path, err)
			return err
		}

		// Process affiliations first, as they may reference lists that haven't been processed yet
		for aff, entries := range affMap {
			// Affiliated entries are added to the ParsedList of the affiliated list
			if affList, exists := parsedListMap[aff]; exists {
				affList.Entries = append(affList.Entries, entries...)
				parsedListMap[aff] = affList
			} else {
				parsedListMap[aff] = model.ParsedList{
					Name:    aff,
					Entries: entries,
				}
			}
		}

		// Add the ParsedList to the map, merging with existing entries if the list already exists
		if existingList, exists := parsedListMap[parsedList.Name]; exists {
			existingList.Entries = append(existingList.Entries, parsedList.Entries...)
			existingList.Inclusions = append(existingList.Inclusions, parsedList.Inclusions...)
			parsedListMap[parsedList.Name] = existingList
		} else {
			parsedListMap[parsedList.Name] = *parsedList
		}

		return nil

	}); err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	return parsedListMap, nil
}

// ParseFile parses a single domain list file and returns a ParsedList along with any affiliations found in the file
func ParseFile(ctx context.Context, path string) (*model.ParsedList, map[string][]model.Entry, error) {
	listName := strings.ToUpper(filepath.Base(path))

	file, err := os.Open(path)
	if err != nil {
		logger.ErrorContext(ctx, "failed to open file: %v", err)
		return nil, nil, err
	}
	defer file.Close()

	var (
		entries        []model.Entry
		inclusions     []model.Inclusion
		affiliationMap = make(map[string][]model.Entry)
	)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			logger.ErrorContext(ctx, "error reading file: %v", err)
			return nil, nil, err
		}
		lineNum++
		line := scanner.Text()
		entry, inc, affs, err := ParseLine(ctx, line)
		if err != nil {
			logger.WarnContext(ctx, " error in %s:%d processing %q - %v", path, lineNum, line, err)
			continue
		}
		if inc != nil {
			inclusions = append(inclusions, *inc)
		}
		if entry != nil {
			entries = append(entries, *entry)
		}
		for _, aff := range affs {
			affiliationMap[aff] = append(affiliationMap[aff], *entry)
		}
	}
	return &model.ParsedList{
		Name:       listName,
		Entries:    entries,
		Inclusions: inclusions,
	}, affiliationMap, nil
}

func ParseLine(ctx context.Context, line string) (*model.Entry, *model.Inclusion, []string, error) {
	line, _, _ = strings.Cut(line, "#")
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		logger.WarnContext(ctx, "skipping empty line")
		return nil, nil, nil, nil
	}
	typeStr, rule, isTyped := strings.Cut(line, ":")
	typeStr = strings.ToLower(typeStr)

	var ruleType model.RuleType
	// default type is domain if not specified
	if !isTyped {
		ruleType = model.TypeDomain
		rule = line
	} else {
		ruleType = model.RuleType(typeStr)
	}

	switch ruleType {
	case model.TypeInclude:
		return parseInclusionRule(ctx, rule)
	case model.TypeDomain, model.TypeFull, model.TypeKeyword, model.TypeRegexp:
		return parseCommonRule(ctx, ruleType, rule)
	default:
		logger.WarnContext(ctx, "unrecognized rule type: %s, skipping line: %q", typeStr, line)
		return nil, nil, nil, fmt.Errorf("unrecognized rule type: %s", typeStr)
	}
}

func parseInclusionRule(ctx context.Context, rule string) (*model.Entry, *model.Inclusion, []string, error) {

	// rule format: `target [@attr1 @attr2 ...]``
	// rule example: `bytedance @-!cn`

	parts := strings.Fields(rule)
	if len(parts) == 0 {
		logger.WarnContext(ctx, "skipping empty inclusion rule")
		return nil, nil, nil, fmt.Errorf("empty rule")
	}

	// target is the first part of the rule
	// it is the ParsedList.Name of the list to include
	// in geosite build process, each file under `data` corresponds to a ParsedList,
	// and the uppercase filename without extension is the ParsedList.Name

	// target example: BYTEDANCE
	target := strings.ToUpper(parts[0])

	var (
		banAttrs, mustAttrs []string
	)
	for _, partStr := range parts[1:] {
		if strings.HasPrefix(partStr, "@") {
			attr := strings.ToLower(partStr[1:])
			if strings.HasPrefix(attr, "-") {
				// banAttrs are marked as `@-<attr>` in the source list, but stored as `<attr>` for easier processing.
				// banAttrs example: `@-!cn` means entries with `!cn` attribute will be excluded from inclusion
				banAttrs = append(banAttrs, attr[1:])
			} else {
				mustAttrs = append(mustAttrs, attr)
			}
		} else if strings.HasPrefix(partStr, "&") {
			logger.WarnContext(ctx, "affiliation tags forbidden on inclusions, skipping line: %q", rule)
			return nil, nil, nil, fmt.Errorf("affiliation tags forbidden on inclusions, rule: %q", rule)
		}
	}

	return nil, &model.Inclusion{
		Target:    target,
		MustAttrs: mustAttrs,
		BanAttrs:  banAttrs,
	}, nil, nil
}

func parseCommonRule(ctx context.Context, ruleType model.RuleType, rule string) (*model.Entry, *model.Inclusion, []string, error) {
	// rule format: `value [@attr1 @attr2 ...]``
	// rule example: `dl.google.com @cn`

	parts := strings.Fields(rule)
	if len(parts) == 0 {
		return nil, nil, nil, fmt.Errorf("empty rule")
	}

	value := parts[0]
	if ruleType == model.TypeDomain || ruleType == model.TypeFull || ruleType == model.TypeKeyword {
		value = strings.ToLower(parts[0])
	}
	var (
		attrs, affiliations []string
	)

	for _, partStr := range parts[1:] {
		if strings.HasPrefix(partStr, "@") {
			attrs = append(attrs, strings.ToLower(partStr[1:]))
		} else if strings.HasPrefix(partStr, "&") {
			affiliations = append(affiliations, strings.ToUpper(partStr[1:]))
		} else {
			logger.WarnContext(ctx, "unrecognized modifier: %s, skipping line: %q", partStr, rule)
			return nil, nil, nil, fmt.Errorf("unrecognized modifier: %s", partStr)
		}
	}
	slices.Sort(attrs)

	return &model.Entry{
		Type:  ruleType,
		Value: value,
		Attrs: attrs,
	}, nil, affiliations, nil
}
