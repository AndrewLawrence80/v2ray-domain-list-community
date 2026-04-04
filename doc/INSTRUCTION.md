# V2Ray Domain List Community - Implementation and Usage Guide

This document provides a detailed overview of the project's purpose, data organization, and internal running mechanism. As a fork of the `v2fly/domain-list-community` project, this workspace aims to maintain and generate `geosite.dat` (or `dlc.dat`) routing databases for V2Ray and compatible clients.

## 1. Project Purpose

The primary purpose of this project is to manage a structured community-driven list of domains and subdomains grouped by entities (e.g., Google, Apple, Microsoft) and categories (e.g., ads, trackers). These lists are compiled and translated into a highly optimized Protocol Buffers (`.dat`) format.

V2Ray (and other compatible proxy software) uses the generated `geosite.dat` file in its routing configuration. By referencing these lists in routing rules (like `geosite:google` or `geosite:ads`), users can efficiently direct traffic to different proxy or direct outbounds based on the domain.

## 2. Data Organization

All raw routing rules are maintained in plaintext files under the `data/` directory.

### 2.1 File Structure
Each file in the `data/` directory represents a specific domain category or list (e.g., `google`, `apple`, `category-ads-all`). The filename itself becomes the list's tag (converted to uppercase internally, but referenced case-insensitively).

### 2.2 Rule Syntax
Inside a data file, each line defines a single domain rule. Comments start with `#` and are ignored.

A rule structure typically follows this format:
`[type:]<value>[@attribute1][&affiliation]`

- **`type`**: The matching type of the domain rule. If omitted, it defaults to `domain`. Supported types:
  - `domain`: Matches the domain and all its subdomains (e.g., `example.com` matches `www.example.com`).
  - `full`: Exact match for the domain (e.g., `full:example.com` will NOT match `www.example.com`).
  - `keyword`: Matches if the given string is part of the target domain.
  - `regexp`: Uses a regular expression to match the domain.
  - `include`: A special type used to include rules from other lists.

- **`value`**: The actual domain, keyword, or regex pattern.

- **`@attribute`**: Optional tags that label specific rules, allowing them to be selectively filtered when included by other lists.
  - E.g., `domain:google.cn @cn` (labels the entry with the `cn` attribute).

- **`&affiliation`**: Optional directive to associate a rule with another list dynamically without writing it twice.
  - E.g., `domain:example.com &google` (dynamically adds `example.com` to the `google` list as well).

### 2.3 Includes and Filtering
You can include rules from another list using the `include` type, reducing duplication. The includes can be paired with attribute filters to selectively merge entries:
- `include:listname`: Includes all entries from `listname`.
- `include:listname@attr`: Includes only entries from `listname` that have the `@attr` attribute.
- `include:listname@-attr`: Includes entries from `listname` explicitly banning those with the `@attr` attribute.

## 3. Running Mechanism

The core generator operates completely automatically without requiring manual interaction with the Protobuf API.

### 3.1 The Main Generator (`main.go`)

Running `go run main.go` executes the generation process, which handles the following steps:
1. **Loading Data**: Iterates through all files inside the `data/` directory. Reads the plaintext rules, parsing the syntax elements (types, attributes, affiliations).
2. **Syntax Validation**: Ensures all list names, domain strings, and attributes only contain valid characters (e.g., letters, numbers, hyphens).
3. **Resolving Cross-Dependencies**: Evaluates all `include:` definitions recursively. It checks for and prevents circular dependency crashes.
4. **List Polishing (Deduplication)**: Consolidates domain listings inside each resolved category. If both a parent domain (`example.com`) and a subdomain (`sub.example.com`) are present without specialized attributes, the redundant subdomain is naturally removed to optimize memory overhead for the proxy client.
5. **Protobuf Compilation**: Transforms the finalized domain lists into V2Ray `router.GeoSite` Proto structures.
6. **Task Assemblies**: Serializes datasets based on configurations. By default, it generates everything into `dlc.dat`, but a JSON profile can be provided via the `-datprofile` flag to define multiple `.dat` files with specific allowlists and denylists of tags.
7. **Export Capabilities**: You can optionally use the `-exportlists` flag followed by comma-separated list names to extract standalone plaintext subsets.

### 3.2 Command-line Arguments

- `-datapath`: Base path to the raw `data/` directory. (Default: `./data`)
- `-outputdir`: Directory where the final `.dat` compilation happens. (Default: `./`)
- `-outputname`: Filename for the generated bundle. (Default: `dlc.dat`)
- `-exportlists`: Lists to be flattened into standalone `-txt` exports (e.g., `-exportlists=google,apple,category-ads-all`).
- `-datprofile`: Optional JSON profile mapping multi-DAT task generations, providing granular `allowlist` and `denylist` mechanisms per dat bundle.

### 3.3 The Dat Dump Tool (`cmd/datdump/main.go`)

Additionally, the project contains a reverse-compiler tool inside `cmd/datdump/`.
If you need to inspect existing `geosite.dat` binaries manually or compare output generation consistency, you can run:

```bash
go run ./cmd/datdump/main.go -inputdata=dlc.dat -outputdir=datdump_out/
```
The datdump parses the protobuf binaries back into human-readable YAML lists. You can use `-exportlists=_all_` to evaluate all sites bundled.

## 4. Internal Domain Organization in Memory (`main.go`)

During the generation process, domains and lists are modeled in memory using the following structures before being compiled into the Protobuf format:

- **`Entry`**: Represents a single parsed domain rule. It stores the `Type` (e.g., `domain`, `full`), `Value` (the domain string), `Attrs` (a slice of attributes), and `Plain` (a deterministic string representation used for deduplication).
- **`Inclusion`**: Captures one `include:` directive. It records the target list (`Source`), required attributes (`MustAttrs`), and forbidden attributes (`BanAttrs`).
- **`ParsedList`**: Represents an entire category file loaded from disk. It holds its `Name`, a list of `Inclusions`, and a list of direct `Entries`.

**Processing Flow (`Processor`)**:
1. **Parsing (`plMap`)**: The generator maps each file to a `ParsedList` via `plMap[string]*ParsedList`. At this stage, domains and includes are simply parsed but not merged.
2. **Resolution (`finalMap`)**: A recursive function flattens the hierarchy and resolves inclusions. Nested elements from included lists are evaluated against `MustAttrs` and `BanAttrs`. The resulting flat list of entries is stored in `finalMap[string][]*Entry`.
3. **Deduplication (`polishList`)**: Before finalization, `polishList` groups entries and removes redundancies. If `example.com` exists as a `domain` rule, any `sub.example.com` entries (without specific attributes) are removed to minimize binary size. Entries are then strictly alphabetically sorted to ensure reproducible builds.
## 5. V2 Architecture Update

The current version of this repository uses a completely renewed `v2` implementation (formerly developed as `v3`) which decouples the generator into granular, context-aware Go packages (`model`, `parser`, `resolver`, `optimizer`, `logger`).

Rather than outputting monolithic Protobuf (`.dat`) structures, the `v2` generator outputs to a highly optimized `geosite.json` that aligns with modern configurable proxy deployments while heavily preventing state-leak bugs by utilizing isolated pointer structures (`resolverState`).

For full details on this new structure, package logic, and how to run it, please refer to the dedicated guide:
[doc/V2_IMPLEMENTATION.md](V2_IMPLEMENTATION.md).