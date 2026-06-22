package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/spf13/cobra"
)

// NewKBCommand creates the kb command for knowledge base operations.
func NewKBCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	kbCmd := &cobra.Command{
		Use:   "kb",
		Short: "Knowledge base operations",
		Long:  "Search, list, add, remove, and manage knowledge base documents and roots",
	}

	// Add subcommands
	kbCmd.AddCommand(NewKBSearchCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBGraphCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBListCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBAddCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBRemoveCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBRootCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBRebuildCacheCommand(log, cfg, t))

	return kbCmd
}

// NewKBSearchCommand creates the kb search subcommand.
func NewKBSearchCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var format string
	var limit int

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search knowledge base documents",
		Long:  "Search knowledge base documents by keywords across all configured roots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			// Get KB roots from config
			roots := cfg.KB.Roots
			if len(roots) == 0 {
				fmt.Println("No knowledge base roots configured")
				return nil
			}

			// Create index path
			indexPath := filepath.Join(config.GetCockpitDir(), ".kb-index.json")

			// Create manager
			manager := kb.NewManager(roots, indexPath)

			// Perform search
			results, err := manager.Search(query)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(results.Results) == 0 {
				fmt.Println("No documents found matching your query")
				return nil
			}

			// Limit results
			if limit > 0 && len(results.Results) > limit {
				results.Results = results.Results[:limit]
				results.Total = limit
			}

			// Output results
			switch format {
			case "json":
				return outputJSON(results)
			case "table":
				return outputTable(results)
			default:
				return outputDefault(results)
			}
		},
	}

	searchCmd.Flags().StringVar(&format, "format", "default", "Output format (default, json, table)")
	searchCmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of results")

	return searchCmd
}

// NewKBListCommand creates the kb list subcommand.
func NewKBListCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var format string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all knowledge base documents",
		Long:  "List all documents in the knowledge base across all configured roots",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get KB roots from config
			roots := cfg.KB.Roots
			if len(roots) == 0 {
				fmt.Println("No knowledge base roots configured")
				return nil
			}

			// Create index path
			indexPath := filepath.Join(config.GetCockpitDir(), ".kb-index.json")

			// Create manager
			manager := kb.NewManager(roots, indexPath)

			// Load all documents
			documents, err := manager.ListDocuments()
			if err != nil {
				return fmt.Errorf("failed to load documents: %w", err)
			}

			if len(documents) == 0 {
				fmt.Println("No documents found in knowledge base")
				return nil
			}

			// Output results
			switch format {
			case "json":
				return outputDocumentsJSON(documents)
			case "table":
				return outputDocumentsTable(documents)
			default:
				return outputDocumentsDefault(documents)
			}
		},
	}

	listCmd.Flags().StringVar(&format, "format", "default", "Output format (default, json, table)")

	return listCmd
}

// NewKBAddCommand creates the kb add subcommand.
func NewKBAddCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add <file>",
		Short: "Add a document to the knowledge base",
		Long:  "Add a new document to the knowledge base",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			// Get KB directory
			kbDir := filepath.Join(config.GetCockpitDir(), "kb")

			// For now, just show a message
			// Full implementation would copy file and validate
			fmt.Printf("Adding document: %s\n", filePath)
			fmt.Printf("To knowledge base: %s\n", kbDir)
			fmt.Println("Feature coming soon...")

			return nil
		},
	}

	return addCmd
}

// NewKBRemoveCommand creates the kb remove subcommand.
func NewKBRemoveCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	removeCmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a document from the knowledge base",
		Long:  "Remove a document from the knowledge base by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// Get KB directory
			kbDir := filepath.Join(config.GetCockpitDir(), "kb")

			// For now, just show a message
			// Full implementation would find and delete document
			fmt.Printf("Removing document: %s\n", id)
			fmt.Printf("From knowledge base: %s\n", kbDir)
			fmt.Println("Feature coming soon...")

			return nil
		},
	}

	return removeCmd
}

// NewKBRootCommand creates the kb root subcommand for managing KB roots.
func NewKBRootCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "root",
		Short: "Manage knowledge base roots",
		Long:  "Add, remove, or list knowledge base root directories",
	}

	// Add subcommands
	rootCmd.AddCommand(NewKBRootAddCommand(log, cfg, t))
	rootCmd.AddCommand(NewKBRootRemoveCommand(log, cfg, t))
	rootCmd.AddCommand(NewKBRootListCommand(log, cfg, t))

	return rootCmd
}

// NewKBRootAddCommand creates the kb root add subcommand.
func NewKBRootAddCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	addCmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a knowledge base root",
		Long:  "Add a new directory as a knowledge base root",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := args[0]

			// Create index path
			indexPath := filepath.Join(config.GetCockpitDir(), ".kb-index.json")

			// Create manager with current roots
			manager := kb.NewManager(cfg.KB.Roots, indexPath)

			// Add root
			err := manager.AddRoot(rootPath)
			if err != nil {
				return fmt.Errorf("failed to add root: %w", err)
			}

			// Update config
			cfg.KB.Roots = manager.GetRoots()
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Added knowledge base root: %s\n", rootPath)
			return nil
		},
	}

	return addCmd
}

// NewKBRootRemoveCommand creates the kb root remove subcommand.
func NewKBRootRemoveCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	removeCmd := &cobra.Command{
		Use:   "remove <path>",
		Short: "Remove a knowledge base root",
		Long:  "Remove a directory from the knowledge base roots",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootPath := args[0]

			// Create index path
			indexPath := filepath.Join(config.GetCockpitDir(), ".kb-index.json")

			// Create manager with current roots
			manager := kb.NewManager(cfg.KB.Roots, indexPath)

			// Remove root
			err := manager.RemoveRoot(rootPath)
			if err != nil {
				return fmt.Errorf("failed to remove root: %w", err)
			}

			// Update config
			cfg.KB.Roots = manager.GetRoots()
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Removed knowledge base root: %s\n", rootPath)
			return nil
		},
	}

	return removeCmd
}

// NewKBRootListCommand creates the kb root list subcommand.
func NewKBRootListCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List knowledge base roots",
		Long:  "List all configured knowledge base root directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			roots := cfg.KB.Roots
			if len(roots) == 0 {
				fmt.Println("No knowledge base roots configured")
				return nil
			}

			fmt.Println("Knowledge Base Roots:")
			for i, root := range roots {
				fmt.Printf("%d. %s\n", i+1, root)
			}

			return nil
		},
	}

	return listCmd
}

// NewKBRebuildCacheCommand creates the kb rebuild-cache subcommand.
func NewKBRebuildCacheCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	rebuildCmd := &cobra.Command{
		Use:   "rebuild-cache",
		Short: "Rebuild the knowledge base index cache",
		Long:  "Rebuild the knowledge base index cache from all configured roots",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get KB roots from config
			roots := cfg.KB.Roots
			if len(roots) == 0 {
				fmt.Println("No knowledge base roots configured")
				return nil
			}

			// Create index path
			indexPath := filepath.Join(config.GetCockpitDir(), ".kb-index.json")

			// Create manager
			manager := kb.NewManager(roots, indexPath)

			// Rebuild index
			fmt.Println("Rebuilding knowledge base index...")
			err := manager.RebuildIndex()
			if err != nil {
				return fmt.Errorf("failed to rebuild index: %w", err)
			}

			// Get last update time
			lastUpdate, err := manager.GetLastIndexUpdate()
			if err != nil {
				return fmt.Errorf("failed to get index update time: %w", err)
			}

			fmt.Println("Knowledge base index rebuilt successfully")
			fmt.Printf("Last updated: %s\n", lastUpdate.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	return rebuildCmd
}

// outputJSON outputs search results as JSON.
func outputJSON(results *kb.SearchResults) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// outputDefault outputs search results in default format.
func outputDefault(results *kb.SearchResults) error {
	fmt.Printf("Search Results for: %q\n", results.Query)
	fmt.Printf("Found: %d documents\n\n", results.Total)

	for i, result := range results.Results {
		fmt.Printf("%d. %s\n", i+1, result.Title)
		fmt.Printf("   ID: %s\n", result.ID)
		fmt.Printf("   Score: %.2f (keyword: %.2f, semantic: %.2f)\n", result.Score, result.KeywordScore, result.SemanticScore)
		fmt.Printf("   Tags: %v\n", result.Tags)
		fmt.Printf("   Excerpt: %s\n", result.Excerpt)
		fmt.Printf("   Path: %s\n\n", result.Path)
	}

	return nil
}

// outputTable outputs search results in table format.
func outputTable(results *kb.SearchResults) error {
	fmt.Printf("Search Results for: %q\n", results.Query)
	fmt.Printf("Found: %d documents\n\n", results.Total)

	// Print header
	fmt.Printf("%-30s | %-6s | %-8s | %-20s\n", "Title", "Score", "Keywords", "Path")
	fmt.Println(string(make([]byte, 80)))

	// Print rows
	for _, result := range results.Results {
		title := result.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		path := result.Path
		if len(path) > 20 {
			path = "..." + path[len(path)-17:]
		}

		fmt.Printf("%-30s | %.2f   | %.2f     | %-20s\n", title, result.Score, result.KeywordScore, path)
	}

	return nil
}

// outputDocumentsJSON outputs documents as JSON.
func outputDocumentsJSON(documents []*kb.Document) error {
	type DocInfo struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Path        string   `json:"path"`
		Tags        []string `json:"tags"`
	}

	var docs []DocInfo
	for _, doc := range documents {
		docs = append(docs, DocInfo{
			ID:          doc.ID,
			Title:       doc.Metadata.Title,
			Description: doc.Metadata.Description,
			Path:        doc.Path,
			Tags:        doc.Metadata.Tags,
		})
	}

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// outputDocumentsDefault outputs documents in default format.
func outputDocumentsDefault(documents []*kb.Document) error {
	fmt.Printf("Knowledge Base Documents (%d total)\n\n", len(documents))

	for i, doc := range documents {
		fmt.Printf("%d. %s\n", i+1, doc.Metadata.Title)
		fmt.Printf("   ID: %s\n", doc.ID)
		fmt.Printf("   Description: %s\n", doc.Metadata.Description)
		fmt.Printf("   Tags: %v\n", doc.Metadata.Tags)
		fmt.Printf("   Path: %s\n\n", doc.Path)
	}

	return nil
}

// outputDocumentsTable outputs documents in table format.
func outputDocumentsTable(documents []*kb.Document) error {
	fmt.Printf("Knowledge Base Documents (%d total)\n\n", len(documents))

	// Print header
	fmt.Printf("%-30s | %-20s | %-30s\n", "Title", "ID", "Path")
	fmt.Println(string(make([]byte, 85)))

	// Print rows
	for _, doc := range documents {
		title := doc.Metadata.Title
		if len(title) > 30 {
			title = title[:27] + "..."
		}

		id := doc.ID
		if len(id) > 20 {
			id = id[:17] + "..."
		}

		path := doc.Path
		if len(path) > 30 {
			path = "..." + path[len(path)-27:]
		}

		fmt.Printf("%-30s | %-20s | %-30s\n", title, id, path)
	}

	return nil
}

// NewKBGraphCommand creates the kb graph subcommand.
func NewKBGraphCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var depth int

	graphCmd := &cobra.Command{
		Use:   "graph <doc-id>",
		Short: "Perform a graph search starting from a document",
		Long:  "Perform a breadth-first search on the knowledge base graph starting from the given document ID.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			docID := args[0]
			roots := cfg.KB.Roots
			if len(roots) == 0 {
				fmt.Println("No knowledge base roots configured")
				return nil
			}

			indexPath := filepath.Join(config.GetCockpitDir(), ".kb-index.json")
			manager := kb.NewManager(roots, indexPath)

			docs, err := manager.ListDocuments()
			if err != nil {
				return fmt.Errorf("failed to list documents: %w", err)
			}

			searcher := kb.NewGraphSearcher()
			if err := searcher.BuildGraph(docs); err != nil {
				return fmt.Errorf("failed to build graph: %w", err)
			}

			res, err := searcher.SearchGraph(docID, depth)
			if err != nil {
				return fmt.Errorf("graph search failed: %w", err)
			}

			fmt.Printf("Graph Search Results for: %q (Depth: %d)\\n", res.RootID, res.MaxDepth)
			fmt.Printf("Found: %d related documents\\n\\n", res.TotalDocs)

			fmt.Printf("%-30s | %-20s | %-8s\\n", "Title", "ID", "Distance")
			fmt.Println(string(make([]byte, 65)))

			for _, node := range res.Nodes {
				title := node.Document.Metadata.Title
				if len(title) > 30 {
					title = title[:27] + "..."
				}
				id := node.Document.ID
				if len(id) > 20 {
					id = id[:17] + "..."
				}
				fmt.Printf("%-30s | %-20s | %d\\n", title, id, node.Distance)
			}

			return nil
		},
	}

	graphCmd.Flags().IntVarP(&depth, "depth", "d", 1, "Maximum depth for graph traversal")
	return graphCmd
}
