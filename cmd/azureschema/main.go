package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/matt-FFFFFF/azureschema/internal/bicep"
	"github.com/matt-FFFFFF/azureschema/internal/render"
	"github.com/urfave/cli/v3"
)

// version is set at build time via ldflags: -X main.version=<version>
var version = "dev"

func main() {
	if err := buildApp().Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func buildApp() *cli.Command {
	return &cli.Command{
		Name:    "azureschema",
		Version: version,
		Usage:   "Query Azure resource type schemas from the command line",
		Description: `Data source: bicep-types-az (https://github.com/Azure/bicep-types-az)

By default, fetches data from GitHub and caches locally.
Use --types-dir to point to a local copy of the bicep-types-az generated directory.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "types-dir",
				Usage: "Path to a local bicep-types-az generated directory (offline mode)",
			},
		},
		Commands: []*cli.Command{
			cmdGet(),
			cmdVersions(),
		},
	}
}

// newSource creates a bicep.Source based on the --types-dir flag.
func newSource(cmd *cli.Command) bicep.Source {
	typesDir := cmd.String("types-dir")
	if typesDir != "" {
		return &bicep.LocalSource{Dir: typesDir}
	}
	return bicep.NewRemoteSource()
}

func cmdGet() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Fetch the schema for a resource type at a given API version",
		ArgsUsage: "<ResourceType> <ApiVersion>",
		Description: `Default output is a human-readable summary. Pass --json for raw resolved JSON.

Examples:
  azureschema get Microsoft.ContainerService/managedClusters 2025-10-01
  azureschema get Microsoft.Storage/storageAccounts 2023-01-01 --json
  azureschema get Microsoft.Storage/storageAccounts 2023-01-01 --depth 3`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output raw resolved JSON instead of summary",
			},
			&cli.IntFlag{
				Name:  "depth",
				Value: 5,
				Usage: "Resolve nested object properties to N levels deep",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 2 {
				return fmt.Errorf("requires exactly 2 arguments: <ResourceType> <ApiVersion>\n\nExample: azureschema get Microsoft.Storage/storageAccounts 2023-01-01")
			}

			resourceType := cmd.Args().Get(0)
			apiVersion := cmd.Args().Get(1)
			jsonOutput := cmd.Bool("json")
			maxDepth := int(cmd.Int("depth"))

			src := newSource(cmd)

			// Load index.
			indexData, err := src.ReadIndex(ctx)
			if err != nil {
				return fmt.Errorf("loading index: %w", err)
			}

			idx, err := bicep.ParseIndexFile(indexData)
			if err != nil {
				return err
			}

			// Look up the resource in the index.
			ref, err := bicep.LookupResource(idx, resourceType, apiVersion)
			if err != nil {
				provider := strings.SplitN(resourceType, "/", 2)[0]
				return fmt.Errorf("%w\nUse 'azureschema versions %s' to list available versions", err, provider)
			}

			// Load the types file.
			typesData, err := src.ReadTypesFile(ctx, ref.FilePath)
			if err != nil {
				return fmt.Errorf("loading types: %w", err)
			}

			types, err := bicep.ParseTypesFile(typesData)
			if err != nil {
				return err
			}

			resolver := bicep.NewResolver(types, maxDepth)

			if jsonOutput {
				return render.JSON(os.Stdout, resolver, ref.TypeIndex)
			}
			return render.Summary(os.Stdout, resolver, ref.TypeIndex, resourceType, apiVersion)
		},
	}
}

func cmdVersions() *cli.Command {
	return &cli.Command{
		Name:      "versions",
		Usage:     "List available API versions for all resource types under a provider",
		ArgsUsage: "<ResourceProvider>",
		Description: `Examples:
  azureschema versions Microsoft.Storage
  azureschema versions Microsoft.ContainerService`,
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() < 1 {
				return fmt.Errorf("requires 1 argument: <ResourceProvider>\n\nExample: azureschema versions Microsoft.Storage")
			}

			provider := cmd.Args().Get(0)
			src := newSource(cmd)

			// Load index.
			indexData, err := src.ReadIndex(ctx)
			if err != nil {
				return fmt.Errorf("loading index: %w", err)
			}

			idx, err := bicep.ParseIndexFile(indexData)
			if err != nil {
				return err
			}

			results := bicep.ListVersions(idx, provider)
			if len(results) == 0 {
				return fmt.Errorf("no resource types found for provider %q", provider)
			}

			// Sort by resource type, then by API version.
			sort.Slice(results, func(i, j int) bool {
				if results[i][0] == results[j][0] {
					return results[i][1] < results[j][1]
				}
				return results[i][0] < results[j][0]
			})

			// Print as a table.
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, r := range results {
				fmt.Fprintf(tw, "%s\t%s\n", r[0], r[1])
			}
			tw.Flush()

			fmt.Fprintf(os.Stderr, "\n%d resource type/version(s) found for %s\n", len(results), provider)
			return nil
		},
	}
}
