package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/scheduler"
	"github.com/spf13/cobra"
)

// NewSchedulerCommand creates the scheduler command.
func NewSchedulerCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	schedulerCmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Gerencia agendamentos de comandos e scripts",
		Long:  "Cria, lista, remove e executa agendamentos de comandos e scripts do AICockpit.",
	}

	schedulerCmd.AddCommand(NewSchedulerAddCommand(log, cfg, t))
	schedulerCmd.AddCommand(NewSchedulerListCommand(log, cfg, t))
	schedulerCmd.AddCommand(NewSchedulerRemoveCommand(log, cfg, t))
	schedulerCmd.AddCommand(NewSchedulerRunCommand(log, cfg, t))
	schedulerCmd.AddCommand(NewSchedulerInstallCommand(log, cfg, t))
	schedulerCmd.AddCommand(NewSchedulerAddUbuntuSecurityCommand(log, cfg, t))

	return schedulerCmd
}

// NewSchedulerAddCommand creates the scheduler add subcommand.
func NewSchedulerAddCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var cronExpr, interval, description string
	var repeat int

	addCmd := &cobra.Command{
		Use:   "add --command <cmd> [--cron <expr> | --interval <interval>]",
		Short: "Adiciona um novo agendamento",
		Long:  `Cria um agendamento para executar um comando periodicamente usando cron ou intervalo fixo.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			command, err := cmd.Flags().GetString("command")
			if err != nil {
				return fmt.Errorf("failed to read command flag: %w", err)
			}

			if strings.TrimSpace(command) == "" {
				return fmt.Errorf("o parametro --command e obrigatorio")
			}

			var jobType string
			if cronExpr != "" {
				jobType = string(scheduler.JobTypeCron)
			} else if interval != "" {
				jobType = string(scheduler.JobTypeRepeat)
			} else {
				return fmt.Errorf("informe --cron ou --interval")
			}

			m := scheduler.NewManager(nil, nil)
			job, err := m.AddJob(command, jobType, cronExpr, interval, repeat, description)
			if err != nil {
				return fmt.Errorf("falha ao criar agendamento: %w", err)
			}

			fmt.Println("[+] Agendamento criado com sucesso!")
			fmt.Printf("    ID: %s\n", job.ID)
			fmt.Printf("    Comando: %s\n", job.Command)
			if jobType == string(scheduler.JobTypeCron) {
				fmt.Printf("    Padrão: %s (%s)\n", job.CronExpr, scheduler.FormatCronDescription(job.CronExpr))
			} else {
				fmt.Printf("    Intervalo: %s\n", job.Interval)
				if job.MaxExecutions > 0 {
					fmt.Printf("    Repetições: %d\n", job.MaxExecutions)
				}
			}
			fmt.Printf("    Próxima execução: %s\n", job.NextRun.Format("2006-01-02 15:04:05"))

			return nil
		},
	}

	addCmd.Flags().String("command", "", "Comando ou script a ser executado")
	addCmd.Flags().StringVar(&cronExpr, "cron", "", "Expressão cron (ex: '0 9 * * *', 'daily', 'weekdays')")
	addCmd.Flags().StringVar(&interval, "interval", "", "Intervalo fixo (ex: 1h, 30m, 1d)")
	addCmd.Flags().IntVar(&repeat, "repeat", 0, "Número máximo de execuções (0 = ilimitado)")
	addCmd.Flags().StringVar(&description, "description", "", "Descrição do agendamento")

	return addCmd
}

// NewSchedulerListCommand creates the scheduler list subcommand.
func NewSchedulerListCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Lista todos os agendamentos",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := scheduler.NewManager(nil, nil)
			jobs, err := m.ListJobs()
			if err != nil {
				return fmt.Errorf("falha ao listar agendamentos: %w", err)
			}

			fmt.Println(scheduler.FormatJobList(jobs))
			return nil
		},
	}

	return listCmd
}

// NewSchedulerRemoveCommand creates the scheduler remove subcommand.
func NewSchedulerRemoveCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	removeCmd := &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove um agendamento",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			m := scheduler.NewManager(nil, nil)
			if err := m.RemoveJob(id); err != nil {
				return fmt.Errorf("falha ao remover agendamento: %w", err)
			}
			fmt.Printf("[-] Agendamento %s removido com sucesso.\n", id)
			return nil
		},
	}

	return removeCmd
}

// NewSchedulerRunCommand creates the scheduler run subcommand.
func NewSchedulerRunCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var all bool

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Executa agendamentos que estão devendo",
		Long:  `Verifica quais agendamentos devem ser executados e os executa. Use --all para forcar todos.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := scheduler.NewManager(nil, nil)

			var err error
			if all {
				err = m.RunAllNow()
			} else {
				err = m.RunDueJobs(time.Now())
			}
			if err != nil {
				return fmt.Errorf("falha ao executar agendamentos: %w", err)
			}
			return nil
		},
	}

	runCmd.Flags().BoolVar(&all, "all", false, "Executa todos os agendamentos ativos imediatamente")

	return runCmd
}

// NewSchedulerInstallCommand creates the scheduler install subcommand.
func NewSchedulerInstallCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var mode string
	var intervalMinutes int

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Instala o mecanismo de execução dos agendamentos",
		Long:  `Instala uma cron job ou um systemd timer para executar 'cockpit scheduler run' periodicamente.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := scheduler.NewManager(nil, nil)

			switch mode {
			case "cron":
				if err := m.InstallCronJob(intervalMinutes); err != nil {
					return fmt.Errorf("falha ao instalar cron: %w", err)
				}
			case "systemd":
				if err := m.InstallSystemdTimer(intervalMinutes); err != nil {
					return fmt.Errorf("falha ao instalar systemd timer: %w", err)
				}
			default:
				return fmt.Errorf("modo invalido: %s (use 'cron' ou 'systemd')", mode)
			}

			return nil
		},
	}

	installCmd.Flags().StringVar(&mode, "mode", "cron", "Modo de instalação: cron ou systemd")
	installCmd.Flags().IntVar(&intervalMinutes, "interval", 5, "Intervalo em minutos entre execuções")

	return installCmd
}

// NewSchedulerAddUbuntuSecurityCommand creates a helper to schedule daily ubuntu-security reports.
func NewSchedulerAddUbuntuSecurityCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var cronExpr string

	addCmd := &cobra.Command{
		Use:   "add-ubuntu-security",
		Short: "Agenda a analise diaria de seguranca do Ubuntu",
		Long:  `Cria um agendamento para executar 'cockpit ubuntu-security report --html' todos os dias.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			m := scheduler.NewManager(nil, nil)

			command := "cockpit ubuntu-security report --html"
			job, err := m.AddJob(command, string(scheduler.JobTypeCron), cronExpr, "", 0, "Analise diaria de seguranca do Ubuntu")
			if err != nil {
				return fmt.Errorf("falha ao criar agendamento: %w", err)
			}

			fmt.Println("[+] Agendamento de seguranca criado com sucesso!")
			fmt.Printf("    ID: %s\n", job.ID)
			fmt.Printf("    Comando: %s\n", job.Command)
			fmt.Printf("    Padrão: %s (%s)\n", job.CronExpr, scheduler.FormatCronDescription(job.CronExpr))
			fmt.Printf("    Próxima execução: %s\n", job.NextRun.Format("2006-01-02 15:04:05"))
			fmt.Println("    Dica: rode 'cockpit scheduler install --mode systemd' para ativar o executor automatico.")
			return nil
		},
	}

	addCmd.Flags().StringVar(&cronExpr, "cron", "0 2 * * *", "Expressao cron para a analise diaria")

	return addCmd
}

// setupSchedulerLog creates a log file for the scheduler run output.
func setupSchedulerLog() (*os.File, error) {
	logDir := filepath.Join(os.Getenv("HOME"), ".cockpit", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}
	logPath := filepath.Join(logDir, "scheduler.log")
	return os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
}
