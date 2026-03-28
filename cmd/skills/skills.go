// Package skillscmd предоставляет CLI команды для управления навыками
package skillscmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/qwen-claw/internal/skills"
	"github.com/user/qwen-claw/internal/skills/security"
)

var (
	skillsDir     string
	verbose       bool
	force         bool
	isolation     string
	auditLogFile  string
)

// NewCommand создаёт команду skills
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Управление навыками",
		Long:  "Просмотр, установка, удаление и управление навыками",
	}

	// Флаги
	cmd.PersistentFlags().StringVar(&skillsDir, "skills-dir", "skills", "директория с навыками")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "подробный вывод")
	cmd.PersistentFlags().StringVar(&isolation, "isolation", "non-main", "уровень изоляции (off/non-main/all)")
	cmd.PersistentFlags().StringVar(&auditLogFile, "audit-log", "", "файл для аудита")

	// Подкоманды
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newInfoCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newInstallCommand())
	cmd.AddCommand(newUninstallCommand())
	cmd.AddCommand(newEnableCommand())
	cmd.AddCommand(newDisableCommand())
	cmd.AddCommand(newSearchCommand())
	cmd.AddCommand(newPopularCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newAuditCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newRegistryCommand())

	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Список навыков",
		RunE: func(cmd *cobra.Command, args []string) error {
			engine := skills.NewEngine(skillsDir)
			if err := engine.Load(); err != nil {
				return fmt.Errorf("failed to load skills: %w", err)
			}

			skillList := engine.List()
			if len(skillList) == 0 {
				fmt.Println("Нет установленных навыков")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tSTATUS\tDESCRIPTION")
			fmt.Fprintln(w, "----\t----\t------\t-----------")

			for _, s := range skillList {
				status := "enabled"
				if !s.Enabled {
					status = "disabled"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Type, status, s.Description)
			}
			w.Flush()

			fmt.Printf("\nВсего: %d навыков\n", len(skillList))
			return nil
		},
	}
}

func newInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info [name]",
		Short: "Информация о навыке",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			skillPath := filepath.Join(skillsDir, name)

			// Проверяем наличие SKILL.md
			skillFile := filepath.Join(skillPath, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				// Пытаемся прочитать skill.json
				jsonFile := filepath.Join(skillPath, "skill.json")
				data, err = os.ReadFile(jsonFile)
				if err != nil {
					return fmt.Errorf("skill not found: %s", name)
				}

				var skill skills.Skill
				if err := json.Unmarshal(data, &skill); err != nil {
					return fmt.Errorf("failed to parse skill.json: %w", err)
				}

				fmt.Printf("Name:        %s\n", skill.Name)
				fmt.Printf("Description: %s\n", skill.Description)
				fmt.Printf("Type:        %s\n", skill.Type)
				fmt.Printf("Commands:    %v\n", skill.Commands)
				fmt.Printf("Enabled:     %v\n", skill.Enabled)
				return nil
			}

			// Парсим YAML frontmatter
			info := parseSkillManifest(string(data))
			printSkillInfo(info)
			return nil
		},
	}
}

func newRunCommand() *cobra.Command {
	var argsList []string

	cmd := &cobra.Command{
		Use:   "run [name] [args...]",
		Short: "Выполнить навык",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if len(args) > 1 {
				argsList = args[1:]
			}

			skillPath := filepath.Join(skillsDir, name)

			// Валидация навыка
			validation, err := security.ValidateSkill(skillPath)
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}
			if !validation.Passed {
				fmt.Printf("⚠️  Предупреждения валидации:\n")
				for _, issue := range validation.Issues {
					fmt.Printf("   - %s\n", issue)
				}
				if !force {
					return fmt.Errorf("skill validation failed. Use --force to run anyway")
				}
			}

			// Читаем манифест
			skillFile := filepath.Join(skillPath, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				return fmt.Errorf("failed to read SKILL.md: %w", err)
			}

			info := parseSkillManifest(string(data))

			// Создаём песочницу
			sandboxConfig := security.DefaultSandboxConfig()
			switch isolation {
			case "off":
				sandboxConfig.Isolation = security.IsolationOff
			case "all":
				sandboxConfig.Isolation = security.IsolationAll
			}
			sandboxConfig.Permissions.Network = info.Permissions.Network
			sandboxConfig.Permissions.Filesystem = info.Permissions.Filesystem

			sandbox := security.NewSandbox(sandboxConfig)

			// Создаём аудитор
			var auditor *security.Auditor
			if auditLogFile != "" {
				auditor = security.NewAuditor(auditLogFile)
			}

			// Определяем entrypoint
			entrypoint := filepath.Join(skillPath, info.Entrypoint)

			// Выполняем навык
			ctx := cmd.Context()
			start := time.Now()

			result, err := sandbox.Run(ctx, entrypoint, argsList, nil)

			duration := time.Since(start)

			// Логируем в аудит
			if auditor != nil {
				auditor.Log(security.AuditEntry{
					Timestamp:      time.Now(),
					SkillName:      name,
					Command:        entrypoint,
					Args:           argsList,
					Success:        err == nil,
					Duration:       duration,
					IsolationLevel: sandboxConfig.Isolation,
					Permissions:    sandboxConfig.Permissions,
				})
			}

			if err != nil {
				return fmt.Errorf("skill execution failed: %w", err)
			}

			fmt.Println(result.Output)
			if result.ExitCode != 0 {
				fmt.Printf("\n⚠️  Exit code: %d\n", result.ExitCode)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "выполнить несмотря на ошибки валидации")

	return cmd
}

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install [name]",
		Short: "Установить навык из реестра",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// TODO: Реализовать установку из реестра
			fmt.Printf("🔧 Установка навыка: %s\n", name)
			fmt.Println("⚠️  Функция установки из реестра в разработке")
			fmt.Println()
			fmt.Println("Вы можете установить навык вручную:")
			fmt.Printf("  git clone https://github.com/sizotov81-hub/qwen-claw-skill-%s.git %s\n", name, filepath.Join(skillsDir, name))
			return nil
		},
	}
}

func newUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [name]",
		Short: "Удалить навык",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			skillPath := filepath.Join(skillsDir, name)

			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				return fmt.Errorf("skill not found: %s", name)
			}

			if err := os.RemoveAll(skillPath); err != nil {
				return fmt.Errorf("failed to remove skill: %w", err)
			}

			fmt.Printf("✅ Навык удалён: %s\n", name)
			return nil
		},
	}
}

func newEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable [name]",
		Short: "Включить навык",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// TODO: Реализовать включение навыка
			fmt.Printf("⚠️  Функция включения в разработке\n")
			fmt.Printf("Навык %s будет включён при следующем запуске\n", name)
			return nil
		},
	}
}

func newDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable [name]",
		Short: "Отключить навык",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// TODO: Реализовать отключение навыка
			fmt.Printf("⚠️  Функция отключения в разработке\n")
			fmt.Printf("Навык %s будет отключён при следующем запуске\n", name)
			return nil
		},
	}
}

func newSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Поиск навыков в реестре",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			// TODO: Реализовать поиск в реестре
			fmt.Printf("🔍 Поиск навыков по запросу: %s\n", query)
			fmt.Println("⚠️  Функция поиска в реестре в разработке")
			return nil
		},
	}
}

func newPopularCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "popular",
		Short: "Популярные навыки",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Реализовать получение популярных навыков
			fmt.Println("⭐ Популярные навыки:")
			fmt.Println("⚠️  Функция в разработке")
			return nil
		},
	}
}

func newUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Проверить обновления навыков",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Реализовать проверку обновлений
			fmt.Println("🔄 Проверка обновлений навыков...")
			fmt.Println("⚠️  Функция в разработке")
			return nil
		},
	}
}

func newAuditCommand() *cobra.Command {
	var limit int
	var skillFilter string

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Показать журнал аудита",
		RunE: func(cmd *cobra.Command, args []string) error {
			if auditLogFile == "" {
				auditLogFile = filepath.Join(os.Getenv("HOME"), ".qwen-claw", "skills-audit.log")
			}

			data, err := os.ReadFile(auditLogFile)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Журнал аудита пуст")
					return nil
				}
				return fmt.Errorf("failed to read audit log: %w", err)
			}

			var entries []security.AuditEntry
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			for _, line := range lines {
				var entry security.AuditEntry
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					continue
				}
				if skillFilter != "" && entry.SkillName != skillFilter {
					continue
				}
				entries = append(entries, entry)
			}

			if len(entries) == 0 {
				fmt.Println("Записей не найдено")
				return nil
			}

			// Сортируем по времени
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Timestamp.After(entries[j].Timestamp)
			})

			// Ограничиваем вывод
			if limit > 0 && len(entries) > limit {
				entries = entries[:limit]
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TIME\tSKILL\tSTATUS\tDURATION\tCOMMAND")
			fmt.Fprintln(w, "----\t-----\t------\t--------\t-------")

			for _, e := range entries {
				status := "✓"
				if !e.Success {
					status = "✗"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n",
					e.Timestamp.Format("2006-01-02 15:04:05"),
					e.SkillName,
					status,
					e.Duration,
					strings.Join(append([]string{e.Command}, e.Args...), " "))
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "максимум записей")
	cmd.Flags().StringVarP(&skillFilter, "skill", "s", "", "фильтр по навыку")

	return cmd
}

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate [name]",
		Short: "Валидировать навык",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			skillPath := filepath.Join(skillsDir, name)

			if _, err := os.Stat(skillPath); os.IsNotExist(err) {
				return fmt.Errorf("skill not found: %s", name)
			}

			result, err := security.ValidateSkill(skillPath)
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			if result.Passed {
				fmt.Printf("✅ Навык %s прошёл валидацию\n", name)
			} else {
				fmt.Printf("❌ Навык %s не прошёл валидацию\n", name)
			}

			if len(result.Issues) > 0 {
				fmt.Println("\nПроблемы:")
				for _, issue := range result.Issues {
					fmt.Printf("  - %s\n", issue)
				}
			}

			return nil
		},
	}
}

func newRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Управление реестром навыков",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Инициализировать локальный реестр",
		RunE: func(cmd *cobra.Command, args []string) error {
			registryDir := filepath.Join(skillsDir, "registry")
			if err := os.MkdirAll(registryDir, 0755); err != nil {
				return fmt.Errorf("failed to create registry dir: %w", err)
			}

			// Создаём index.json
			indexFile := filepath.Join(registryDir, "index.json")
			index := map[string]interface{}{
				"version":   "1.0.0",
				"skills":    []string{},
				"updated":   time.Now().Format(time.RFC3339),
			}

			data, _ := json.MarshalIndent(index, "", "  ")
			if err := os.WriteFile(indexFile, data, 0644); err != nil {
				return fmt.Errorf("failed to create index: %w", err)
			}

			fmt.Printf("✅ Локальный реестр инициализирован: %s\n", registryDir)
			return nil
		},
	})

	return cmd
}

// SkillInfo информация из манифеста
type SkillInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	License     string
	Type        string
	Entrypoint  string
	Commands    []string
	Permissions struct {
		Network    bool
		Filesystem security.FilesystemAccess
		Elevated   bool
	}
	Dependencies []string
	Tags         []string
}

// parseSkillManifest парсит YAML frontmatter
func parseSkillManifest(content string) SkillInfo {
	var info SkillInfo

	// Простой парсинг YAML frontmatter
	lines := strings.Split(content, "\n")
	inFrontmatter := false

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				break
			}
		}

		if !inFrontmatter {
			continue
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "version:") {
			info.Version = strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		} else if strings.HasPrefix(line, "description:") {
			info.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		} else if strings.HasPrefix(line, "author:") {
			info.Author = strings.TrimSpace(strings.TrimPrefix(line, "author:"))
		} else if strings.HasPrefix(line, "type:") {
			info.Type = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
		} else if strings.HasPrefix(line, "entrypoint:") {
			info.Entrypoint = strings.TrimSpace(strings.TrimPrefix(line, "entrypoint:"))
		} else if strings.HasPrefix(line, "commands:") {
			// Пропускаем, обрабатываем отдельно
		} else if strings.HasPrefix(line, "- ") && len(info.Commands) > 0 {
			info.Commands = append(info.Commands, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
		}
	}

	// Отдельно парсим commands
	if idx := strings.Index(content, "commands:"); idx != -1 {
		rest := content[idx:]
		lines := strings.Split(rest, "\n")
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "- ") {
				info.Commands = append(info.Commands, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			} else if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "#") {
				break
			}
		}
	}

	return info
}

// printSkillInfo выводит информацию о навыке
func printSkillInfo(info SkillInfo) {
	fmt.Printf("Name:         %s\n", info.Name)
	fmt.Printf("Version:      %s\n", info.Version)
	fmt.Printf("Description:  %s\n", info.Description)
	fmt.Printf("Author:       %s\n", info.Author)
	fmt.Printf("License:      %s\n", info.License)
	fmt.Printf("Type:         %s\n", info.Type)
	fmt.Printf("Entrypoint:   %s\n", info.Entrypoint)
	fmt.Printf("Commands:     %s\n", strings.Join(info.Commands, ", "))
	fmt.Printf("Network:      %v\n", info.Permissions.Network)
	fmt.Printf("Filesystem:   %s\n", info.Permissions.Filesystem)
	fmt.Printf("Elevated:     %v\n", info.Permissions.Elevated)

	if len(info.Dependencies) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(info.Dependencies, ", "))
	}

	if len(info.Tags) > 0 {
		fmt.Printf("Tags:         %s\n", strings.Join(info.Tags, ", "))
	}
}
