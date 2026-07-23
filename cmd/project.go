package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/project"
	"github.com/spf13/cobra"
)

func NewProjectCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var (
		projectTitle string
		projectDesc  string
	)

	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Gerencia projetos e seus boards",
		Long:  `Cria, gerencia, acompanha tasks e organiza repositórios/pastas de projetos.`,
	}

	getProjectManager := func() *project.Manager {
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, ".cockpit", "workspace", "projects")
		return project.NewManager(path)
	}

	projectCreateCmd := &cobra.Command{
		Use:   "create <slug>",
		Short: "Cria um novo projeto",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if projectTitle == "" {
				projectTitle = slug
			}

			mgr := getProjectManager()
			proj, err := mgr.CreateProject(slug, projectTitle, projectDesc)
			if err != nil {
				return err
			}

			fmt.Printf("✅ Projeto '%s' criado com sucesso em: %s\n", proj.Metadata.Title, proj.Path)
			return nil
		},
	}
	projectCreateCmd.Flags().StringVar(&projectTitle, "title", "", "Título do projeto")
	projectCreateCmd.Flags().StringVar(&projectDesc, "description", "", "Descrição do projeto")

	projectListCmd := &cobra.Command{
		Use:   "list",
		Short: "Lista todos os projetos",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			projects, err := mgr.ListProjects()
			if err != nil {
				return err
			}

			if len(projects) == 0 {
				fmt.Println("Nenhum projeto encontrado.")
				return nil
			}

			fmt.Println("📦 Projetos:")
			for _, p := range projects {
				fmt.Printf(" - %s (%s) [%d tasks, %d repos]\n", p.Metadata.Title, p.ID, len(p.Metadata.Tasks), len(p.Metadata.Repositories))
			}
			return nil
		},
	}

	projectInfoCmd := &cobra.Command{
		Use:   "info <slug>",
		Short: "Exibe os metadados do projeto",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("📂 Projeto: %s (%s)\n", proj.Metadata.Title, proj.ID)
			fmt.Printf("Descrição: %s\n", proj.Metadata.Description)
			fmt.Printf("Tags: %v\n", proj.Metadata.Tags)
			fmt.Printf("Tasks: %d registradas\n", len(proj.Metadata.Tasks))
			fmt.Printf("Repos: %d vinculados\n", len(proj.Metadata.Repositories))
			fmt.Printf("Caminho: %s\n", proj.Path)
			return nil
		},
	}

	projectTagAddCmd := &cobra.Command{
		Use:   "tag-add <slug> <tag>",
		Short: "Adiciona uma tag ao projeto",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}

			for _, t := range proj.Metadata.Tags {
				if t == args[1] {
					fmt.Println("Tag já existe.")
					return nil
				}
			}

			proj.Metadata.Tags = append(proj.Metadata.Tags, args[1])
			if err := mgr.SaveProject(proj); err != nil {
				return err
			}
			fmt.Printf("✅ Tag '%s' adicionada.\n", args[1])
			return nil
		},
	}

	// Task subcommands
	projectTaskCmd := &cobra.Command{Use: "task", Short: "Gerencia as tasks do board"}

	projectTaskCmd.AddCommand(&cobra.Command{
		Use:   "add <slug> \"Descrição\"",
		Short: "Adiciona uma task à coluna todo",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getProjectManager().AddTask(args[0], args[1]); err != nil {
				return err
			}
			fmt.Println("✅ Task adicionada com sucesso.")
			return nil
		},
	})

	projectTaskCmd.AddCommand(&cobra.Command{
		Use:   "move <slug> <task-id> <coluna>",
		Short: "Move uma task para outra coluna",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getProjectManager().MoveTask(args[0], args[1], args[2]); err != nil {
				return err
			}
			fmt.Printf("✅ Task movida para a coluna '%s'.\n", args[2])
			return nil
		},
	})

	projectTaskCmd.AddCommand(&cobra.Command{
		Use:   "reorder <slug> <task-id> <new-index>",
		Short: "Muda a posição de uma task dentro do board",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("new-index must be an integer")
			}
			if err := getProjectManager().ReorderTask(args[0], args[1], idx); err != nil {
				return err
			}
			fmt.Println("✅ Task reordenada com sucesso.")
			return nil
		},
	})

	projectTaskCmd.AddCommand(&cobra.Command{
		Use:   "list <slug>",
		Short: "Lista todas as tasks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			proj, err := getProjectManager().GetProject(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("📌 Board do projeto '%s':\n", proj.ID)
			for _, col := range proj.Metadata.BoardColumns {
				fmt.Printf("\n[%s]\n", strings.ToUpper(col))
				count := 0
				for _, task := range proj.Metadata.Tasks {
					if task.Status == col {
						fmt.Printf(" - %s: %s\n", task.ID, task.Title)
						count++
					}
				}
				if count == 0 {
					fmt.Println("   (Vazio)")
				}
			}
			return nil
		},
	})

	projectTaskCmd.AddCommand(&cobra.Command{
		Use:   "sync <slug> <task-id>",
		Short: "Sincroniza a task com uma issue no GitHub",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := getProjectManager().SyncGitHubIssue(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Printf("✅ Task sincronizada com sucesso. Issue #%d\n", task.IssueNumber)
			return nil
		},
	})

	// Board
	projectBoardCmd := &cobra.Command{Use: "board", Short: "Configurações do Kanban Board"}
	projectBoardCmd.AddCommand(&cobra.Command{
		Use:   "column-add <slug> <nome>",
		Short: "Adiciona uma nova coluna ao board",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}
			proj.Metadata.BoardColumns = append(proj.Metadata.BoardColumns, args[1])
			if err := mgr.SaveProject(proj); err != nil {
				return err
			}
			fmt.Printf("✅ Coluna '%s' adicionada.\n", args[1])
			return nil
		},
	})

	// Repo
	projectRepoCmd := &cobra.Command{Use: "repo", Short: "Gerencia repositórios"}
	projectRepoCmd.AddCommand(&cobra.Command{
		Use:  "add <slug> <url>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}
			proj.Metadata.Repositories = append(proj.Metadata.Repositories, args[1])
			if err := mgr.SaveProject(proj); err != nil {
				return err
			}
			fmt.Println("✅ Repositório adicionado.")
			return nil
		},
	})

	// Workspace
	projectWorkspaceCmd := &cobra.Command{Use: "workspace", Short: "Gerencia caminhos locais"}
	projectWorkspaceCmd.AddCommand(&cobra.Command{
		Use:  "add <slug> <path>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}
			absPath, _ := filepath.Abs(args[1])
			proj.Metadata.Workspaces = append(proj.Metadata.Workspaces, absPath)
			if err := mgr.SaveProject(proj); err != nil {
				return err
			}
			fmt.Println("✅ Workspace local adicionado.")
			return nil
		},
	})

	// KB
	projectKbCmd := &cobra.Command{Use: "kb", Short: "Associa KBs ao projeto"}
	projectKbCmd.AddCommand(&cobra.Command{
		Use:  "link <slug> <kb-slug>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}
			proj.Metadata.KnowledgeBases = append(proj.Metadata.KnowledgeBases, args[1])
			if err := mgr.SaveProject(proj); err != nil {
				return err
			}
			fmt.Println("✅ KB vinculado.")
			return nil
		},
	})

	// Links
	projectLinkCmd := &cobra.Command{Use: "link", Short: "Associa links externos ao projeto"}
	projectLinkCmd.AddCommand(&cobra.Command{
		Use:  "add <slug> <url> <title>",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getProjectManager()
			proj, err := mgr.GetProject(args[0])
			if err != nil {
				return err
			}
			proj.Metadata.Links = append(proj.Metadata.Links, project.Link{
				URL:   args[1],
				Title: args[2],
			})
			if err := mgr.SaveProject(proj); err != nil {
				return err
			}
			fmt.Println("✅ Link externo adicionado.")
			return nil
		},
	})

	projectTrackCmd := &cobra.Command{
		Use:   "track <slug> \"Mensagem\"",
		Short: "Registra uma anotação de andamento",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getProjectManager().AddTracking(args[0], args[1]); err != nil {
				return err
			}
			fmt.Println("✅ Log registrado com sucesso.")
			return nil
		},
	}

	projectCmd.AddCommand(projectCreateCmd)
	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectInfoCmd)
	projectCmd.AddCommand(projectTagAddCmd)
	projectCmd.AddCommand(projectTrackCmd)

	projectCmd.AddCommand(projectTaskCmd)
	projectCmd.AddCommand(projectBoardCmd)
	projectCmd.AddCommand(projectRepoCmd)
	projectCmd.AddCommand(projectWorkspaceCmd)
	projectCmd.AddCommand(projectKbCmd)
	projectCmd.AddCommand(projectLinkCmd)

	return projectCmd
}
