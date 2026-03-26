# azureschema

A CLI tool for querying Azure resource type schemas from the command line.

`azureschema` consumes the [bicep-types-az](https://github.com/Azure/bicep-types-az) data source -- the same type definitions used by Azure Bicep -- and lets you look up the full schema (properties, types, flags, descriptions) for any Azure resource type at a specific API version.

## Installation

### From release binaries

Download the latest release for your platform from the [Releases](https://github.com/matt-FFFFFF/azureschema/releases) page. Binaries are available for Linux, macOS, and Windows on both amd64 and arm64.

### From source

```bash
go install github.com/matt-FFFFFF/azureschema/cmd/azureschema@latest
```

Or clone and build locally:

```bash
git clone https://github.com/matt-FFFFFF/azureschema.git
cd azureschema
go build -o azureschema ./cmd/azureschema
```

## Usage

### Get a resource schema

Fetch the full schema for a resource type at a specific API version:

```bash
azureschema get Microsoft.Storage/storageAccounts 2023-01-01
```

This outputs a human-readable summary with property trees, type annotations, and flags.

#### JSON output

```bash
azureschema get Microsoft.Storage/storageAccounts 2023-01-01 --json
```

#### Control resolution depth

Nested type references are resolved recursively up to a configurable depth (default: 5):

```bash
azureschema get Microsoft.Storage/storageAccounts 2023-01-01 --depth 3
```

### List available API versions

List all resource types and API versions for a provider:

```bash
azureschema versions Microsoft.Storage
```

### Offline mode

If you have a local clone of [bicep-types-az](https://github.com/Azure/bicep-types-az), point to the `generated/` directory to avoid network requests:

```bash
azureschema --types-dir /path/to/bicep-types-az/generated get Microsoft.Storage/storageAccounts 2023-01-01
```

## Features

- **Online and offline data sources** -- Fetches schemas from GitHub by default, with optional local directory support for air-gapped or faster usage.
- **Local caching** -- Remote data is cached at `~/.cache/azure-schema/` (respects `XDG_CACHE_HOME`). The resource index is cached with a 24-hour TTL; type definitions are cached permanently.
- **Recursive type resolution** -- Follows type references through the schema, resolving nested objects, arrays, unions, and literal types into a complete tree.
- **Human-readable output** -- Summary view with Unicode box-drawing, indented property trees, type annotations (e.g. `string`, `array<string>`, `("Active" | "Inactive" | string)`), and property flags (`[REQUIRED]`, `[READ-ONLY]`, `[WRITE-ONLY]`).
- **JSON output** -- Fully resolved JSON tree for scripting and automation.
- **Case-insensitive lookup** -- Resource type names are matched case-insensitively when an exact match is not found.

## Development

### Prerequisites

- Go 1.26+

### Run tests

```bash
go test ./...
```

### Project structure

```
cmd/azureschema/       CLI entry point and integration tests
internal/bicep/        Data model, index loading, source interface, type resolver
internal/render/       Output renderers (summary and JSON)
```

## License

See [LICENSE](LICENSE) for details.
