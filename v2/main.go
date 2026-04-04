package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/v2fly/domain-list-community/v2/logger"
	"github.com/v2fly/domain-list-community/v2/model"
	"github.com/v2fly/domain-list-community/v2/optimizer"
	"github.com/v2fly/domain-list-community/v2/parser"
	"github.com/v2fly/domain-list-community/v2/resolver"
)

var (
	dataPath   = flag.String("data", "../data", "Path to data directory")
	outputPath = flag.String("output", "./geosite.json", "Output JSON path")
)

func run(ctx context.Context) error {
	fmt.Printf("Walking data paths at %s...\n", *dataPath)
	rawMap, err := parser.ParseDirectory(ctx, *dataPath)
	if err != nil {
		return fmt.Errorf("parse dir: %w", err)
	}

	fmt.Println("Resolving domains...")
	entries, err := resolver.Resolve(ctx, rawMap)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}

	fmt.Println("Optimizing subsets...")
	optimizedLists := make(map[string][]model.Entry)
	for name, entries := range entries {
		optimizedLists[strings.ToLower(name)] = optimizer.Optimize(ctx, entries)
	}

	fmt.Printf("Dumping to %s...\n", *outputPath)
	if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.Create(*outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(optimizedLists); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}
	fmt.Println("Done!")
	return nil
}

func main() {
	flag.Parse()
	logger.Init("stdout", "info", "text")
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
