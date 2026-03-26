# azureschema - Go CLI Tool Plan

## Overview

Convert the `azure-schema` bash script into a Go CLI tool (`azureschema`) that queries Azure resource type schemas from the bicep-types-az data source.

## Project Structure

```
azureschema/
├── go.mod                        # github.com/matt-FFFFFF/azureschema
├── go.sum
├── cmd/
│   └── azureschema/
│       └── main.go               # Entry point, urfave/cli app setup
└── internal/
    ├── bicep/
    │   ├── types.go              # Data model structs for bicep-types-az JSON
    │   ├── index.go              # Index loading, lookup, ref resolution
    │   ├── resolver.go           # Recursive type resolution with depth limit
    │   └── source.go             # Data source interface + implementations
    ├── cache/
    │   └── cache.go              # File-based cache (~/.cache/azure-schema, 24h TTL for index)
    └── render/
        ├── json.go               # JSON output renderer
        └── summary.go            # Human-readable summary renderer
```

## Key Design Decisions

1. **Module**: `github.com/matt-FFFFFF/azureschema`
2. **Binary**: `azureschema` (built from `cmd/azureschema/main.go`)
3. **CLI Framework**: `github.com/urfave/cli/v3` (latest version)
4. **Data source abstraction**: A `Source` interface with two implementations:
   - `RemoteSource` - fetches from GitHub raw URLs, caches to `~/.cache/azure-schema`
   - `LocalSource` - reads directly from a local directory (the `--types-dir` flag)
5. **Caching**: Preserved from bash script - `index.json` cached for 24h, `types.json` files cached permanently. Only applies to `RemoteSource`.

## CLI Commands & Flags

```
azureschema get <ResourceType> <ApiVersion> [--json] [--depth N] [--types-dir DIR]
azureschema versions <ResourceProvider> [--types-dir DIR]
azureschema help
```

- `--types-dir` is a global flag (applies to all commands). When set, skips GitHub fetch and reads directly from that directory (a local clone of `bicep-types-az/generated/`).
- `--json` flag on `get` outputs resolved JSON instead of the summary.
- `--depth N` flag on `get` controls recursive resolution depth (default: 5).

## Data Model (internal/bicep/types.go)

The bicep-types-az `types.json` is a JSON array of type objects. Each object has a `$type` discriminator field. Key types to model:

- `ResourceType` - top-level resource entry with `body.$ref`
- `ObjectType` - has `name`, `properties` map (each with `type.$ref`, `flags`, `description`)
- `StringType`, `StringLiteralType`, `IntegerType`, `BooleanType`, `AnyType`
- `ArrayType` - has `itemType.$ref`
- `UnionType` - has `elements[]` with `$ref`

Unmarshalled using a discriminated union pattern (unmarshal into `json.RawMessage` first, peek at `$type`, then unmarshal into the specific struct).

## Type Resolution (internal/bicep/resolver.go)

Mirrors the bash script's jq `resolve` function:
- Takes a type index and recursively resolves `$ref` pointers
- Respects `--depth` limit
- Produces either a structured Go object (for JSON rendering) or feeds the summary renderer

## Renderers

- **JSON renderer** (`internal/render/json.go`): Produces the same JSON structure as the bash script's `render_json` function - resolves types into `{type, properties, items, oneOf, const, required, readOnly, writeOnly, description, ...}`.
- **Summary renderer** (`internal/render/summary.go`): Produces the same box-drawing character layout with `━`, `───`, indented property trees, flags like `[REQUIRED]`, `[READ-ONLY]`, `[WRITE-ONLY]`, and truncated descriptions.

## Bitfield Flags

Same as bash script:
- Bit 0 (1): Required
- Bit 1 (2): ReadOnly
- Bit 2 (4): WriteOnly

## Flow

1. **`get` command**: Load index -> resolve ref (file path + type index) -> load types.json -> resolve body type recursively -> render (JSON or summary)
2. **`versions` command**: Load index -> filter keys by provider prefix (case-insensitive) -> sort and display as table
