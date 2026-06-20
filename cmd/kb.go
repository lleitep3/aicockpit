package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/kb"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/spf13/cobra"
)

// NewKBCommand creates the kb command for knowledge base operations.
func NewKBCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	kbCmd := &cobra.Command{
		Use:   "kb",
		Short: "Knowledge base operations",
		Long:  "Search, list, add, and manage knowledge base documents",
	}

	// Add subcommands
	kbCmd.AddCommand(NewKBSearchCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBListCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBAddCommand(log, cfg, t))
	kbCmd.AddCommand(NewKBRemoveCommand(log, cfg, t))

	return kbCmd
}

// NewKBSearchCommand creates the kb search subcommand.
func NewKBSearchCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var format string
	var limit int

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search knowledge base documents",
		Long:  "Search knowledge base documents by keywords",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			// Get KB directory
			kbDir := filepath.Join(config.GetCockpitDir(), "kb")

			// Create repository and searcher
			repo := kb.NewFileRepository(kbDir)
			scorer := kb.NewDefaultScorer()
			searcher := kb.NewKeywordSearcher(scorer)

			// Load all documents
			documents, err := repo.ListDocuments(".")
			if err != nil {
				return fmt.Errorf("failed to load documents: %w", err)
			}

			if len(documents) == 0 {
				fmt.Println("No documents found in knowledge base")
				return nil
			}

			// Perform search
			results, err := searcher.Search(query, documents)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
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
		Long:  "List all documents in the knowledge base",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get KB directory
			kbDir := filepath.Join(config.GetCockpitDir(), "kb")

			// Create repository
			repo := kb.NewFileRepository(kbDir)

			// Load all documents
			documents, err := repo.ListDocuments(".")
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
