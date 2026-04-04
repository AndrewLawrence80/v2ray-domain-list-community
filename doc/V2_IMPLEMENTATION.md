# V2 Implementation Guide

## Overview

The `v2` implementation of the V2Ray Domain List Community project introduces a highly modular, thread-safe, and robustly tracked architecture. Moving away from the monolithic and slightly stateful structures of previous generator scripts, `v2` completely decouples the generation pipeline into dedicated structural packages with robust logging and Context (`context.Context`) injection.

This ensures the system is not only generating optimized outputs but is highly maintainable natively in modern Go standards.

## Architecture & Packages

The `v2` source code consists of five main internally partitioned sub-packages.

### 1. `model` Package
Serves as the Single Source of Truth for schema definitions. It drops the business logic and instead models purely structural data shapes used across the runtime pipeline:
- **`RuleType`**: Enumerated constants referencing types (`domain`, `full`, `keyword`, `regexp`, `include`).
- **`Entry`**: Handles an individual rule line and exposes a fast, deterministic `.Hash()` method.
- **`Inclusion`**: Defines an inclusion rule with conditions (`Target`, `MustAttrs`, `BanAttrs`).
- **`ParsedList`**: Represents a single file group consisting of inclusions and raw entries.

### 2. `parser` Package
Responsible for scraping the plaintext structure mappings (`data/` directory) and lifting it into memory mapped structures.
- Parses rules dynamically and validates formatting line-by-line via `bufio.Scanner`.
- Resolves cross-file `&affiliation` tokens correctly during the parsing step, inserting the entries straight into multiple affiliated `ParsedList` elements.
- Accepts a `context.Context` for gracefully bounding or canceling the parser tree walk.

### 3. `resolver` Package
Flattens the raw dependencies resolved by the parser. This package processes all `include:` definitions across datasets.
- Fully thread-safe design. Eliminates the previous implementation's risks of leaky package-level caching by wrapping its logic into standalone instantiation bounds.
- Tracks recursive cycles via a DFS algorithm mapped over the tracking dictionaries preventing infinite loops upon circular inclusions.
- Strict evaluation logic verifying whether an entry honors an inclusion's `@<attr>` logic or its `@-<attr>` ban barriers.

### 4. `optimizer` Package
Ensures the binary distribution sizes compress optimally.
- Trims redundant rules naturally covered by broader scoping parent filters—for instance, pruning `domain:www.example.com` automatically when `domain:example.com` already envelopes it natively.
- Selectively operates on non-attributed generic domains (ignoring complex `regexp` masks preserving exact matching limits).
- Provides final deterministically reproducible builds by leveraging the specific struct `Hash()` during slice sorts.

### 5. `logger` Package
A centralized observability package utilizing modern Go context injection patterns allowing clean text logs or structured JSON traces per step (Warning about empty lines, flagging unrecognized rules without abruptly panicking).

---

## Running the V2 Pipeline

Executing the program defaults natively to picking up `../data` and exporting `geosite.json`.

```bash
cd v2
go run . -data="../data" -output="./geosite.json"
```

The system executes pipeline components sequentially:
1. `parser.ParseDirectory(ctx, ...)`
2. `resolver.Resolve(ctx, rawMap)`
3. `optimizer.Optimize(ctx, entries)`
4. Serializes natively optimized dictionaries explicitly directly into the output stream.
