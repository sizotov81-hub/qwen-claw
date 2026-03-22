package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/qwen-claw/internal/agent"
	"github.com/user/qwen-claw/internal/config"
	"github.com/user/qwen-claw/internal/gateway"
	"github.com/user/qwen-claw/internal/logger"
	"github.com/user/qwen-claw/internal/markdown"
	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/sandbox"
	"github.com/user/qwen-claw/internal/scheduler"
	"github.com/user/qwen-claw/internal/skills"
	"github.com/user/qwen-claw/internal/web"
	"github.com/user/qwen-claw/bots/telegram"
)

var (
	cfgFile     string
	verbose     bool
	interactive bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "qwen-claw",
		Short: "Qwen-Claw - оболочка для Qwen Code CLI с расширенными возможностями",
		Long: `Qwen-Claw - это оболочка вокруг Qwen Code CLI, которая добавляет:
- Персистентную память и контекст между сессиями
- Интеграцию с мессенджерами (Telegram, Discord)
- Планировщик задач
- Веб-интерфейс
- Расширяемую систему навыков

Все запросы выполняются через Qwen Code CLI.`,
		RunE: runMain,
	}

	// Флаги
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "путь к файлу конфигурации (yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "подробный вывод")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", false, "интерактивный режим")

	// Команда run
	var runCmd = &cobra.Command{
		Use:   "run [query]",
		Short: "Выполнить запрос через Qwen Code CLI",
		Long:  "Отправить запрос в Qwen Code CLI и получить ответ",
		RunE:  runQuery,
	}
	rootCmd.AddCommand(runCmd)

	// Команда ask
	var askCmd = &cobra.Command{
		Use:   "ask [query]",
		Short: "Задать вопрос (алиас для run)",
		RunE:  runQuery,
	}
	rootCmd.AddCommand(askCmd)

	// Команда exec
	var execCmd = &cobra.Command{
		Use:   "exec [command]",
		Short: "Выполнить shell команду через Qwen",
		RunE:  runQuery,
	}
	rootCmd.AddCommand(execCmd)

	// Команда memory
	var memoryCmd = &cobra.Command{
		Use:   "memory",
		Short: "Управление памятью",
		Long:  "Показать содержимое памяти",
		RunE:  showMemory,
	}
	memoryCmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Очистить память",
		RunE:  clearMemory,
	})
	memoryCmd.AddCommand(&cobra.Command{
		Use:   "search [query]",
		Short: "Поиск в памяти",
		RunE:  searchMemory,
	})
	memoryCmd.AddCommand(&cobra.Command{
		Use:   "add [text]",
		Short: "Добавить запись в память",
		RunE:  addMemory,
	})
	rootCmd.AddCommand(memoryCmd)

	// Команда chat
	var chatCmd = &cobra.Command{
		Use:   "chat",
		Short: "Начать чат-сессию с Qwen",
		Long:  "Интерактивная чат-сессия с сохранением истории",
		RunE:  runChat,
	}
	rootCmd.AddCommand(chatCmd)

	// Команда init
	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Инициализировать проект",
		Long:  "Создать конфигурацию и дирекории",
		RunE:  initProject,
	}
	rootCmd.AddCommand(initCmd)

	// Команда doctor
	var doctorCmd = &cobra.Command{
		Use:   "doctor",
		Short: "Проверить установку",
		Long:  "Проверить доступность Qwen Code CLI и конфигурацию",
		RunE:  doctorCheck,
	}
	rootCmd.AddCommand(doctorCmd)

	// Команда telegram
	var telegramCmd = &cobra.Command{
		Use:   "telegram",
		Short: "Запустить Telegram бота",
		Long:  "Запуск Telegram бота для взаимодействия с Qwen Code CLI",
		RunE:  runTelegramBot,
	}
	rootCmd.AddCommand(telegramCmd)

	// Команда gateway
	var gatewayCmd = &cobra.Command{
		Use:   "gateway",
		Short: "Запустить WebSocket Gateway",
		Long:  "Запуск WebSocket Gateway для управления сессиями и клиентами",
		RunE:  runGateway,
	}
	gatewayCmd.Flags().Int("port", 18789, "порт для прослушивания")
	gatewayCmd.Flags().String("bind", "127.0.0.1", "адрес привязки")
	gatewayCmd.Flags().Bool("debug", false, "режим отладки")
	rootCmd.AddCommand(gatewayCmd)

	// Команда sandbox
	var sandboxCmd = &cobra.Command{
		Use:   "sandbox",
		Short: "Управление Docker Sandbox",
		Long:  "Управление изолированными контейнерами для выполнения команд",
		RunE:  sandboxStatus,
	}
	sandboxCmd.AddCommand(&cobra.Command{
		Use:   "run [command]",
		Short: "Выполнить команду в sandbox",
		RunE:  sandboxRun,
	})
	sandboxCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Список sandbox",
		RunE:  sandboxList,
	})
	sandboxCmd.AddCommand(&cobra.Command{
		Use:   "clean",
		Short: "Очистить старые sandbox",
		RunE:  sandboxClean,
	})
	rootCmd.AddCommand(sandboxCmd)

	// Команда skills
	var skillsCmd = &cobra.Command{
		Use:   "skills",
		Short: "Управление навыками",
		Long:  "Показать список навыков или управлять ими",
		RunE:  listSkills,
	}
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "enable [name]",
		Short: "Включить навык",
		RunE:  enableSkill,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "disable [name]",
		Short: "Отключить навык",
		RunE:  disableSkill,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "run [name] [args...]",
		Short: "Выполнить навык",
		RunE:  runSkill,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "info [name]",
		Short: "Показать информацию о навыке",
		RunE:  skillInfo,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "install [name]",
		Short: "Установить навык из реестра",
		RunE:  installSkill,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "uninstall [name]",
		Short: "Удалить навык",
		RunE:  uninstallSkill,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "search [query]",
		Short: "Поиск навыков в реестре",
		RunE:  searchSkills,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "popular",
		Short: "Популярные навыки",
		RunE:  popularSkills,
	})
	skillsCmd.AddCommand(&cobra.Command{
		Use:   "update",
		Short: "Проверить обновления навыков",
		RunE:  updateSkills,
	})
	rootCmd.AddCommand(skillsCmd)

	// Команда tasks
	var tasksCmd = &cobra.Command{
		Use:   "tasks",
		Short: "Управление задачами планировщика",
		Long:  "Показать список задач или управлять ими",
		RunE:  listTasks,
	}
	tasksCmd.AddCommand(&cobra.Command{
		Use:   "add [name] [schedule] [command]",
		Short: "Добавить задачу",
		Long:  "Добавить задачу с именем, расписанием и командой\nПример: tasks add backup '@daily' 'Сделай резервную копию проекта'",
		RunE:  addTask,
	})
	tasksCmd.AddCommand(&cobra.Command{
		Use:   "remove [id]",
		Short: "Удалить задачу",
		RunE:  removeTask,
	})
	tasksCmd.AddCommand(&cobra.Command{
		Use:   "enable [id]",
		Short: "Включить задачу",
		RunE:  enableTask,
	})
	tasksCmd.AddCommand(&cobra.Command{
		Use:   "disable [id]",
		Short: "Отключить задачу",
		RunE:  disableTask,
	})
	tasksCmd.AddCommand(&cobra.Command{
		Use:   "run [id]",
		Short: "Выполнить задачу немедленно",
		RunE:  runTaskNow,
	})
	tasksCmd.AddCommand(&cobra.Command{
		Use:   "results",
		Short: "Показать результаты выполнения",
		RunE:  taskResults,
	})
	rootCmd.AddCommand(tasksCmd)

	// Команда web
	var webCmd = &cobra.Command{
		Use:   "web",
		Short: "Запустить Web UI",
		Long:  "Запуск веб-интерфейса Qwen-Claw с безопасной аутентификацией",
		RunE:  runWebUI,
	}
	webCmd.Flags().String("host", "127.0.0.1", "хост для прослушивания")
	webCmd.Flags().Int("port", 64656, "порт для прослушивания (по умолчанию 64656)")
	rootCmd.AddCommand(webCmd)

	// Команда degradation
	var degradationCmd = &cobra.Command{
		Use:   "degradation",
		Short: "Управление системой антидеградации",
		Long:  "Просмотр отчётов и управление системой антидеградации",
		RunE:  degradationReport,
	}
	degradationCmd.AddCommand(&cobra.Command{
		Use:   "report",
		Short: "Показать отчёт о деградации",
		RunE:  degradationReport,
	})
	degradationCmd.AddCommand(&cobra.Command{
		Use:   "clear",
		Short: "Очистить историю деградации",
		RunE:  degradationClear,
	})
	rootCmd.AddCommand(degradationCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMain(cmd *cobra.Command, args []string) error {
	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Создаём директории
	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Инициализируем память
	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	// Создаём планировщик с заглушкой executor (будет заменён позже)
	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), nil)
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	// Загружаем навыки с автозагрузкой
	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	// Получаем список навыков с автозагрузкой
	autoLoadSkills := skillEngine.GetAutoLoadSkills()
	if len(autoLoadSkills) > 0 {
		fmt.Printf("🔹 Auto-loading skills (%d):\n", len(autoLoadSkills))
		for _, s := range autoLoadSkills {
			fmt.Printf("   ✅ %s (%s)\n", s.Name, s.Description)
		}
		fmt.Println()
	}

	// Создаём агента
	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			QwenPath:     "", // автоматически найдётся в PATH
			Model:        cfg.LLM.Model,
			ApprovalMode: "auto-edit",
			Timeout:      300000000000, // 5 минут
			Debug:        verbose,
		},
		memoryManager,
	)

	// Устанавливаем планировщик в агент
	agentInstance.SetScheduler(sched)

	// Настраиваем executor для планировщика
	taskExec := agent.NewAgentExecutor(agentInstance)
	sched.SetExecutor(taskExec)

	// Проверяем доступность qwen cli
	if !agentInstance.CheckQwenAvailable() {
		fmt.Fprintf(os.Stderr, "Warning: Qwen Code CLI not found at %s\n", agentInstance.GetQwenPath())
		fmt.Fprintf(os.Stderr, "Please install it: npm install -g @anthropic-ai/qwen-code\n\n")
	}

	// Интерактивный режим
	if interactive || len(args) == 0 {
		return runInteractive(agentInstance)
	}

	// Выполняем запрос из аргументов
	query := strings.Join(args, " ")
	response, err := agentInstance.Run(cmd.Context(), query)
	if err != nil {
		return fmt.Errorf("agent failed: %w", err)
	}

	// Форматируем ответ с markdown для CLI
	formatted := markdown.RenderToCLI(response)
	fmt.Println(formatted)
	return nil
}

func runQuery(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("query is required")
	}

	query := strings.Join(args, " ")

	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: "auto-edit",
			Timeout:      300000000000,
			Debug:        verbose,
		},
		memoryManager,
	)

	if verbose {
		fmt.Printf("Using Qwen CLI: %s\n", agentInstance.GetQwenPath())
		fmt.Printf("Model: %s\n", agentInstance.GetModel())
		fmt.Printf("Query: %s\n\n", query)
	}

	response, err := agentInstance.Run(cmd.Context(), query)
	if err != nil {
		return fmt.Errorf("agent failed: %w", err)
	}

	// Форматируем ответ с markdown для CLI
	formatted := markdown.RenderToCLI(response)
	fmt.Println(formatted)
	return nil
}

func runChat(cmd *cobra.Command, args []string) error {
	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: "auto-edit",
			Timeout:      600000000000, // 10 минут
			Debug:        verbose,
		},
		memoryManager,
	)

	fmt.Println("Qwen-Claw Chat Mode")
	fmt.Println("Commands:")
	fmt.Println("  /help     - показать помощь")
	fmt.Println("  /clear    - очистить историю")
	fmt.Println("  /history  - показать историю сессии")
	fmt.Println("  /memory   - показать память")
	fmt.Println("  /context  - показать контекст")
	fmt.Println("  /exit     - выйти")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n🔹> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Обрабатываем команды
		if strings.HasPrefix(input, "/") {
			switch input {
			case "/help":
				fmt.Println("Available commands:")
				fmt.Println("  /help     - показать помощь")
				fmt.Println("  /clear    - очистить историю")
				fmt.Println("  /history  - показать историю сессии")
				fmt.Println("  /compact  - сжать контекст (суммаризация)")
				fmt.Println("  /memory   - показать память")
				fmt.Println("  /context  - показать контекст")
				fmt.Println("  /exit     - выйти")
				fmt.Println()
				fmt.Println("Примеры запросов:")
				fmt.Println("  exec ls -la          - выполнить команду")
				fmt.Println("  read file.txt        - прочитать файл")
				fmt.Println("  search pattern       - поиск по файлам")
				fmt.Println("  remember fact        - сохранить в память")

			case "/clear":
				agentInstance.ClearHistory()
				fmt.Println("✅ История сессии очищена")

			case "/compact":
				// Выполняем компaction сессии
				events := agentInstance.GetSessionEvents()
				if len(events) == 0 {
					fmt.Println("📋 Нечего сжимать — история пуста")
				} else {
					// Генерируем суммаризацию
					summary := memory.SimpleSummaryGenerator(events)
					if err := agentInstance.CompactSession(summary); err != nil {
						fmt.Printf("❌ Ошибка сжатия: %v\n", err)
					} else {
						fmt.Println("✅ Контекст сжат")
						fmt.Println(summary)
					}
				}

			case "/history":
				// Пробуем получить события из event store
				events := agentInstance.GetSessionEvents()
				if len(events) > 0 {
					// Показываем события с суммаризацией
					summary, recentEvents := agentInstance.GetSessionWithSummary()
					
					if summary != "" {
						fmt.Println("📝 Суммаризация:")
						fmt.Println(summary)
						fmt.Println()
					}
					
					fmt.Printf("📋 История сессии (%d записей, последние %d):\n\n", len(events), len(recentEvents))
					for i, event := range recentEvents {
						if i >= 20 { // Показываем максимум 20
							fmt.Printf("... и ещё %d записей\n", len(recentEvents)-20)
							break
						}
						
						emoji := "💬"
						switch event.Type {
						case memory.EventUserMessage:
							emoji = "👤"
						case memory.EventAssistantReply:
							emoji = "🤖"
						case memory.EventToolCall:
							emoji = "🔧"
						case memory.EventError:
							emoji = "❌"
						}
						
						fmt.Printf("%s [%s] %s\n", emoji, event.Type, event.Content[:100])
						if len(event.Content) > 100 {
							fmt.Printf("   ...%s\n", event.Content[100:])
						}
					}
					
					// Показываем статус компaction
					status := agentInstance.GetCompactionStatus()
					fmt.Printf("\n%s\n", status.FormatStatus())
				} else {
					// Fallback на старую историю
					history := agentInstance.GetSessionHistory()
					if len(history) == 0 {
						fmt.Println("📋 История сессии пуста")
					} else {
						fmt.Printf("📋 История сессии (%d записей):\n\n", len(history))
						for i, msg := range history {
							if i >= 20 {
								fmt.Printf("... и ещё %d записей\n", len(history)-20)
								break
							}
							if strings.HasPrefix(msg, "User:") {
								fmt.Printf("👤 %s\n", msg[5:])
							} else if strings.HasPrefix(msg, "Assistant:") {
								fmt.Printf("🤖 %s\n", msg[10:])
							} else {
								fmt.Printf("   %s\n", msg)
							}
						}
					}
				}

			case "/memory":
				entries := memoryManager.ListEntries()
				if len(entries) == 0 {
					fmt.Println("📚 Память пуста")
				} else {
					fmt.Printf("📚 Память (%d записей):\n\n", len(entries))
					for _, e := range entries {
						fmt.Printf("[%s] %s\n", e.Type, e.Content)
					}
				}

			case "/context":
				ctx := agentInstance.GetMemoryContext()
				if ctx == "" {
					fmt.Println("📭 Контекст пуст")
				} else {
					fmt.Println(ctx)
				}

			case "/exit", "/quit":
				fmt.Println("Goodbye!")
				return nil

			default:
				fmt.Printf("❌ Неизвестная команда: %s\n", input)
			}
			continue
		}

		// Выполняем запрос
		if verbose {
			fmt.Println("\n⏳ Thinking...")
		}

		response, err := agentInstance.Run(cmd.Context(), input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			continue
		}

		// Форматируем ответ с markdown для CLI
		formatted := markdown.RenderToCLI(response)
		fmt.Println("\n💬 " + formatted)
	}

	return scanner.Err()
}

func runInteractive(agentInstance *agent.Agent) error {
	fmt.Println("Qwen-Claw Interactive Mode")
	fmt.Println("Commands:")
	fmt.Println("  /help     - показать помощь")
	fmt.Println("  /clear    - очистить историю")
	fmt.Println("  /memory   - показать память")
	fmt.Println("  /exit     - выйти")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Обрабатываем команды
		if strings.HasPrefix(input, "/") {
			switch input {
			case "/help":
				fmt.Println("Available commands:")
				fmt.Println("  /help     - показать помощь")
				fmt.Println("  /clear    - очистить историю")
				fmt.Println("  /memory   - показать память")
				fmt.Println("  /exit     - выйти")

			case "/clear":
				agentInstance.ClearHistory()
				fmt.Println("History cleared")

			case "/memory":
				fmt.Println("Use 'qwen-claw memory' to see memory details")

			case "/exit", "/quit":
				fmt.Println("Goodbye!")
				return nil

			default:
				fmt.Printf("Unknown command: %s\n", input)
			}
			continue
		}

		// Выполняем запрос
		response, err := agentInstance.Run(context.Background(), input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println(response)
	}

	return scanner.Err()
}

func showMemory(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	entries := memoryManager.ListEntries()
	if len(entries) == 0 {
		fmt.Println("Memory is empty")
		return nil
	}

	fmt.Printf("Memory entries (%d):\n\n", len(entries))
	for _, e := range entries {
		fmt.Printf("[%s] %s\n", e.Type, e.Content)
		fmt.Printf("    Created: %s\n", e.Created.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

func clearMemory(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	if err := memoryManager.Clear(); err != nil {
		return fmt.Errorf("failed to clear memory: %w", err)
	}

	fmt.Println("Memory cleared")
	return nil
}

func searchMemory(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("search query is required")
	}

	query := strings.Join(args, " ")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	results := memoryManager.Recall(query, 20)
	if len(results) == 0 {
		fmt.Println("Nothing found")
		return nil
	}

	fmt.Printf("Found %d entries:\n\n", len(results))
	for _, e := range results {
		fmt.Printf("[%s] %s\n", e.Type, e.Content)
		fmt.Println()
	}

	return nil
}

func addMemory(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("text is required")
	}

	text := strings.Join(args, " ")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	_, err = memoryManager.Remember("fact", text, nil)
	if err != nil {
		return fmt.Errorf("failed to add to memory: %w", err)
	}

	fmt.Println("Added to memory")
	return nil
}

func initProject(cmd *cobra.Command, args []string) error {
	cfg := config.Default()

	// Создаём директории
	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Сохраняем конфигурацию
	configPath := cfgFile
	if configPath == "" {
		configPath = "config.yaml"
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Project initialized at %s\n", cfg.BaseDir)
	fmt.Printf("Config saved to %s\n", configPath)
	fmt.Println()
	fmt.Println("Edit config.yaml to set your model and other settings.")
	fmt.Println("Run 'qwen-claw doctor' to check installation.")
	fmt.Println("Run 'qwen-claw -i' to start interactive mode.")

	return nil
}

func doctorCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("Qwen-Claw Doctor")
	fmt.Println("================")
	fmt.Println()

	allOk := true

	// Проверка 1: Qwen Code CLI
	fmt.Print("1. Checking Qwen Code CLI... ")
	qwenPath, err := exec.LookPath("qwen")
	if err != nil {
		fmt.Println("❌ Not found")
		fmt.Println("   Install with: npm install -g @anthropic-ai/qwen-code")
		allOk = false
	} else {
		fmt.Printf("✓ Found at %s\n", qwenPath)
	}

	// Проверка 2: Конфигурация
	fmt.Print("2. Checking configuration... ")
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Printf("⚠️  Using defaults (%v)\n", err)
	} else {
		fmt.Println("✓ Loaded")
	}

	// Проверка 3: Директории
	fmt.Print("3. Checking directories... ")
	if err := cfg.EnsureDirs(); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		allOk = false
	} else {
		fmt.Println("✓ Created/verified")
	}

	// Проверка 4: Память
	fmt.Print("4. Checking memory... ")
	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		fmt.Printf("❌ Failed: %v\n", err)
		allOk = false
	} else {
		fmt.Println("✓ OK")
	}

	fmt.Println()
	if allOk {
		fmt.Println("✅ All checks passed!")
	} else {
		fmt.Println("⚠️  Some checks failed. Please fix the issues above.")
	}

	return nil
}

func runGateway(cmd *cobra.Command, args []string) error {
	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Получаем флаги
	port, _ := cmd.Flags().GetInt("port")
	bind, _ := cmd.Flags().GetString("bind")
	debug, _ := cmd.Flags().GetBool("debug")

	// Создаём конфигурацию Gateway
	gatewayConfig := &gateway.Config{
		Enabled: true,
		Port:    port,
		Bind:    bind,
		Debug:   debug,
		Auth: gateway.AuthConfig{
			Mode:      "token",
			Password:  "",
			Token:     "", // будет сгенерирован автоматически
			AllowList: []string{},
		},
		Tailscale: gateway.TailscaleConfig{
			Mode: "none",
		},
		MaxClients: 100,
	}

	// Создаём Gateway
	gw, err := gateway.NewGateway(gatewayConfig)
	if err != nil {
		return fmt.Errorf("failed to create gateway: %w", err)
	}

	// Показываем информацию
	fmt.Println("🔌 Qwen-Claw Gateway")
	fmt.Println("===================")
	fmt.Printf("📡 WebSocket: ws://%s:%d/ws\n", bind, port)
	fmt.Printf("🏥 Health:   http://%s:%d/health\n", bind, port)
	fmt.Printf("🔑 Auth Token: %s\n", gw.GetAuthToken())
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  /api/v1/sessions  - список сессий")
	fmt.Println("  /api/v1/clients   - список клиентов")
	fmt.Println("  /api/v1/broadcast - рассылка событий")
	fmt.Println()
	fmt.Println("WebSocket Message Types:")
	fmt.Println("  auth       - аутентификация")
	fmt.Println("  chat       - сообщение чата")
	fmt.Println("  command    - команда")
	fmt.Println("  subscribe  - подписка на канал")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")

	// Настраиваем обработчик сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем Gateway в горутине
	go func() {
		if err := gw.Start(); err != nil {
			logger.Errorf("Gateway error: %v", err)
		}
	}()

	// Ждём сигнал остановки
	<-sigChan
	fmt.Println("\n👋 Stopping Gateway...")

	return gw.Stop()
}

func runTelegramBot(cmd *cobra.Command, args []string) error {
	logger.Info("Loading configuration...")
	
	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// НЕ логируем токен и чувствительные данные!
	logger.Infof("Telegram enabled: %v", cfg.Telegram.Enabled)
	if len(cfg.Telegram.AllowedUsers) > 0 {
		logger.Infof("Access restricted to %d user(s)", len(cfg.Telegram.AllowedUsers))
	} else {
		logger.Info("Access: open for all users")
	}

	// Проверяем токен
	if cfg.Telegram.Token == "" {
		return fmt.Errorf("telegram token is not set in config\n" +
			"Please add your bot token to config.yaml:\n" +
			"telegram:\n" +
			"  enabled: true\n" +
			"  token: \"YOUR_BOT_TOKEN\"")
	}

	// Инициализируем память
	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	// Создаём агента
	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Timeout:      300000000000, // 5 минут
			Debug:        cfg.LLM.Debug,
		},
		memoryManager,
	)

	// Проверяем доступность qwen cli
	if !agentInstance.CheckQwenAvailable() {
		logger.Infof("Warning: Qwen Code CLI not found at %s", agentInstance.GetQwenPath())
	}

	// Создаём планировщик
	taskExec := &taskExecutor{agent: agentInstance}
	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), taskExec)
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}
	sched.Start()
	defer sched.Stop()

	// Создаём бота
	bot, err := telegram.NewBot(
		telegram.BotConfig{
			Token:        cfg.Telegram.Token,
			AllowedUsers: cfg.Telegram.AllowedUsers,
			Timeout:      300000000000, // 5 минут
		},
		agentInstance,
		memoryManager,
		sched,
	)
	if err != nil {
		return fmt.Errorf("failed to create bot: %w", err)
	}

	// Логируем конфигурацию доступа
	if len(cfg.Telegram.AllowedUsers) > 0 {
		logger.Infof("Access restricted to users: %v", cfg.Telegram.AllowedUsers)
	} else {
		logger.Info("Access: open for all users")
	}

	// Показываем информацию о боте
	botInfo, err := bot.GetBotInfo()
	if err != nil {
		logger.Infof("Warning: failed to get bot info: %v", err)
	} else {
		logger.Infof("Bot username: %s", botInfo)
	}

	logger.Info("Starting Telegram bot...")
	logger.Info("Press Ctrl+C to stop")

	// Запускаем бота
	return bot.Start()
}

// listSkills показывает список всех навыков
func listSkills(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	// Получаем навыки из injector
	skillList, err := skillEngine.ListSkills()
	if err != nil {
		return fmt.Errorf("failed to list skills: %w", err)
	}

	if len(skillList) == 0 {
		fmt.Println("📭 No skills found")
		fmt.Println("Install skills with: qwen-claw skills search <query>")
		fmt.Println("Or browse popular: qwen-claw skills popular")
		return nil
	}

	fmt.Printf("📦 Installed Skills (%d total):\n\n", len(skillList))
	for _, s := range skillList {
		status := "✅"
		if !s.Enabled {
			status = "❌"
		}
		fmt.Printf("%s %s\n", status, s.Name)
		fmt.Printf("   %s\n", s.Description)
		if len(s.Commands) > 0 {
			fmt.Printf("   Commands: %s\n", strings.Join(s.Commands, ", "))
		}
		fmt.Println()
	}

	fmt.Println("💡 Commands:")
	fmt.Println("  qwen-claw skills search <query>  - Search skills")
	fmt.Println("  qwen-claw skills install <name>  - Install skill")
	fmt.Println("  qwen-claw skills enable/disable  - Enable/disable skill")
	fmt.Println("  qwen-claw skills update          - Check for updates")

	return nil
}

// enableSkill включает навык
func enableSkill(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skill name is required")
	}

	skillName := args[0]
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	if err := skillEngine.EnableSkill(skillName); err != nil {
		return fmt.Errorf("failed to enable skill: %w", err)
	}

	fmt.Printf("✅ Skill '%s' enabled!\n", skillName)
	return nil
}

// disableSkill отключает навык
func disableSkill(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skill name is required")
	}

	skillName := args[0]
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	if err := skillEngine.DisableSkill(skillName); err != nil {
		return fmt.Errorf("failed to disable skill: %w", err)
	}

	fmt.Printf("✅ Skill '%s' disabled!\n", skillName)
	return nil
}

// runSkill выполняет навык
func runSkill(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skill name is required")
	}

	skillName := args[0]
	skillArgs := args[1:]

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	skill := skillEngine.Get(skillName)
	if skill == nil {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	if !skill.Enabled {
		return fmt.Errorf("skill '%s' is disabled", skillName)
	}

	result, err := skillEngine.Execute(cmd.Context(), &skills.SkillRequest{
		Command: skillName,
		Args:    skillArgs,
	})
	if err != nil {
		return fmt.Errorf("skill execution failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("skill failed: %s", result.Error)
	}

	fmt.Println(result.Output)
	return nil
}

// skillInfo показывает информацию о навыке
func skillInfo(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skill name is required")
	}

	skillName := args[0]
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	skill := skillEngine.Get(skillName)
	if skill == nil {
		return fmt.Errorf("skill '%s' not found", skillName)
	}

	fmt.Printf("📦 Skill: %s\n", skill.Name)
	fmt.Printf("   Description: %s\n", skill.Description)
	fmt.Printf("   Type: %s\n", skill.Type)
	fmt.Printf("   Entry Point: %s\n", skill.EntryPoint)
	fmt.Printf("   Enabled: %v\n", skill.Enabled)
	fmt.Printf("   Commands: %s\n", strings.Join(skill.Commands, ", "))
	return nil
}

// installSkill устанавливает навык из реестра
func installSkill(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skill name is required")
	}

	skillName := args[0]
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	fmt.Printf("📦 Installing skill '%s'...\n", skillName)
	if err := skillEngine.InstallSkill(skillName); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Printf("✅ Skill '%s' installed successfully!\n", skillName)
	return nil
}

// uninstallSkill удаляет навык
func uninstallSkill(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("skill name is required")
	}

	skillName := args[0]
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	fmt.Printf("📦 Uninstalling skill '%s'...\n", skillName)
	if err := skillEngine.UninstallSkill(skillName); err != nil {
		return fmt.Errorf("failed to uninstall skill: %w", err)
	}

	fmt.Printf("✅ Skill '%s' uninstalled successfully!\n", skillName)
	return nil
}

// searchSkills ищет навыки в реестре
func searchSkills(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = strings.Join(args, " ")
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	fmt.Println("🔍 Searching skills...")
	skills, err := skillEngine.SearchSkills(query, "", nil)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found")
		return nil
	}

	fmt.Printf("📦 Found %d skills:\n\n", len(skills))
	for _, s := range skills {
		fmt.Printf("• %s\n", s.Name)
		fmt.Printf("  %s\n", s.Description)
		fmt.Printf("  Category: %s | Downloads: %d | Rating: %s\n", s.Category, s.Downloads, s.Version)
		fmt.Println()
	}

	return nil
}

// popularSkills показывает популярные навыки
func popularSkills(cmd *cobra.Command, args []string) error {
	limit := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &limit)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	fmt.Println("🔥 Popular skills:")
	skills, err := skillEngine.ListPopularSkills(limit)
	if err != nil {
		return fmt.Errorf("failed to get popular skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No skills found")
		return nil
	}

	for i, s := range skills {
		fmt.Printf("%2d. %s - %s\n", i+1, s.Name, s.Description)
	}

	return nil
}

// updateSkills проверяет обновления
func updateSkills(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	skillEngine := skills.NewEngine(cfg.SkillsDir)
	if err := skillEngine.Load(); err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	fmt.Println("🔄 Checking for skill updates...")
	updates, err := skillEngine.CheckForUpdates()
	if err != nil {
		return fmt.Errorf("update check failed: %w", err)
	}

	if len(updates) == 0 {
		fmt.Println("✅ All skills are up to date!")
		return nil
	}

	fmt.Printf("📦 Updates available (%d):\n", len(updates))
	for _, name := range updates {
		fmt.Printf("  • %s\n", name)
	}
	fmt.Println("\nTo update, run: qwen-claw skills install <skill-name>")

	return nil
}

// taskExecutor реализует интерфейс scheduler.Executor
type taskExecutor struct {
	agent *agent.Agent
}

func (e *taskExecutor) Execute(ctx context.Context, command string) (string, error) {
	return e.agent.Run(ctx, command)
}

// listTasks показывает список всех задач
func listTasks(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	tasks := sched.ListTasks()
	if len(tasks) == 0 {
		fmt.Println("No tasks scheduled")
		return nil
	}

	fmt.Printf("📋 Scheduled Tasks (%d total):\n\n", len(tasks))
	for _, t := range tasks {
		status := "✅"
		if !t.Enabled {
			status = "❌"
		}
		fmt.Printf("%s [%s] %s\n", status, t.ID, t.Name)
		fmt.Printf("   Description: %s\n", t.Description)
		fmt.Printf("   Command: %s\n", t.Command)
		fmt.Printf("   Schedule: %s\n", t.Schedule)
		if t.NextRun != nil {
			fmt.Printf("   Next Run: %s\n", t.NextRun.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("   Run Count: %d\n", t.RunCount)
		fmt.Println()
	}

	return nil
}

// addTask добавляет новую задачу
func addTask(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: tasks add <name> <schedule> <command>\nExample: tasks add backup '@daily' 'Сделай резервную копию проекта'")
	}

	name := args[0]
	schedule := args[1]
	command := strings.Join(args[2:], " ")

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	task, err := sched.AddTask(name, name, command, schedule)
	if err != nil {
		return fmt.Errorf("failed to add task: %w", err)
	}

	fmt.Printf("✅ Task added:\n")
	fmt.Printf("   ID: %s\n", task.ID)
	fmt.Printf("   Name: %s\n", task.Name)
	fmt.Printf("   Schedule: %s\n", task.Schedule)
	fmt.Printf("   Command: %s\n", task.Command)
	if task.NextRun != nil {
		fmt.Printf("   Next Run: %s\n", task.NextRun.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()
	fmt.Println("Use 'qwen-claw tasks' to see all tasks.")

	return nil
}

// removeTask удаляет задачу
func removeTask(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	taskID := args[0]

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	if err := sched.RemoveTask(taskID); err != nil {
		return fmt.Errorf("failed to remove task: %w", err)
	}

	fmt.Printf("Task %s removed\n", taskID)
	return nil
}

// enableTask включает задачу
func enableTask(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	taskID := args[0]

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	if err := sched.EnableTask(taskID); err != nil {
		return fmt.Errorf("failed to enable task: %w", err)
	}

	fmt.Printf("Task %s enabled\n", taskID)
	return nil
}

// disableTask отключает задачу
func disableTask(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	taskID := args[0]

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	if err := sched.DisableTask(taskID); err != nil {
		return fmt.Errorf("failed to disable task: %w", err)
	}

	fmt.Printf("Task %s disabled\n", taskID)
	return nil
}

// runTaskNow выполняет задачу немедленно
func runTaskNow(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("task ID is required")
	}

	taskID := args[0]

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	result, err := sched.RunTaskNow(taskID)
	if err != nil {
		return fmt.Errorf("failed to run task: %w", err)
	}

	fmt.Printf("Task executed:\n")
	fmt.Printf("   Success: %v\n", result.Success)
	fmt.Printf("   Duration: %v\n", result.Completed.Sub(result.Started))
	if result.Error != "" {
		fmt.Printf("   Error: %s\n", result.Error)
	}
	if result.Output != "" {
		fmt.Printf("   Output:\n%s\n", result.Output)
	}

	return nil
}

// taskResults показывает результаты выполнения задач
func taskResults(cmd *cobra.Command, args []string) error {
	limit := 10
	if len(args) > 0 {
		if _, err := fmt.Sscanf(args[0], "%d", &limit); err != nil {
			limit = 10
		}
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), &taskExecutor{agent: agentInstance})
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}

	results := sched.GetResults(limit)
	if len(results) == 0 {
		fmt.Println("No task results yet")
		return nil
	}

	fmt.Printf("📊 Task Results (last %d):\n\n", len(results))
	for _, r := range results {
		status := "✅"
		if !r.Success {
			status = "❌"
		}
		fmt.Printf("%s Task: %s\n", status, r.TaskID)
		fmt.Printf("   Duration: %v\n", r.Completed.Sub(r.Started))
		if r.Error != "" {
			fmt.Printf("   Error: %s\n", r.Error)
		}
		if r.Output != "" {
			fmt.Printf("   Output: %s\n", r.Output)
		}
		fmt.Println()
	}

	return nil
}

// runWebUI запускает веб-интерфейс
func runWebUI(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")

	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Инициализируем безопасность Web UI
	webConfig := &web.ServerConfig{
		Host: host,
		Port: port,
	}
	configPath := cfgFile
	if configPath == "" {
		configPath = "config.yaml"
	}
	if err := web.InitSecurity(configPath, webConfig); err != nil {
		return fmt.Errorf("failed to init security: %w", err)
	}

	logger.Infof("🔒 Secure Web UI starting at http://%s:%d", host, port)
	logger.Infof("📝 Use secret phrase for authentication (see .web-secrets.json)")
	logger.Info("Press Ctrl+C to stop")

	// Инициализируем память
	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	// Создаём агента
	agentInstance := agent.NewAgent(
		agent.AgentConfig{
			Model:        cfg.LLM.Model,
			ApprovalMode: cfg.LLM.ApprovalMode,
			Debug:        verbose,
		},
		memoryManager,
	)

	// Создаём планировщик
	taskExec := &taskExecutor{agent: agentInstance}
	sched := scheduler.NewScheduler(filepath.Join(cfg.QwenDir, "scheduler"), taskExec)
	if err := sched.Init(); err != nil {
		return fmt.Errorf("failed to init scheduler: %w", err)
	}
	sched.Start()
	defer sched.Stop()

	// Создаём Gateway для интеграции с Web UI
	gwConfig := &gateway.Config{
		Enabled: true,
		Port:    18789,
		Bind:    "127.0.0.1",
		Debug:   verbose,
		Auth: gateway.AuthConfig{
			Mode:     "token",
			Token:    "", // будет сгенерирован автоматически
			Password: "",
		},
		MaxClients: 100,
	}
	gw, err := gateway.NewGateway(gwConfig)
	if err != nil {
		logger.Warnf("⚠️  Failed to create Gateway: %v", err)
	} else {
		// Запускаем Gateway в горутине
		go func() {
			if err := gw.Start(); err != nil {
				logger.Errorf("Gateway error: %v", err)
			}
		}()
		logger.Info("🔌 Gateway started on :18789")
	}

	// Создаём и запускаем веб-сервер
	webServer := web.NewServer(
		*webConfig,
		agentInstance,
		memoryManager,
		sched,
		gw,
	)

	// Запускаем сервер в горутине
	go func() {
		logger.Info("🌐 Web UI starting...")
		if err := webServer.Start(); err != nil {
			logger.Errorf("Web UI error: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 Graceful shutdown...")

	// Останавливаем веб-сервер с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := webServer.Stop(ctx); err != nil {
		logger.Errorf("Web UI shutdown error: %v", err)
	}

	logger.Info("✅ Web UI stopped")
	return nil
}

// sandboxStatus показывает статус sandbox
func sandboxStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("🔒 Qwen-Claw Sandbox")
	fmt.Println("====================")
	fmt.Println()
	
	// Проверяем доступность Docker
	if !isDockerAvailable() {
		fmt.Println("⚠️  Docker недоступен")
		fmt.Println()
		fmt.Println("Для работы sandbox необходим Docker.")
		fmt.Println("Установите: https://docs.docker.com/get-docker/")
		return nil
	}
	
	fmt.Println("✅ Docker доступен")
	fmt.Println()
	fmt.Println("Команды:")
	fmt.Println("  qwen-claw sandbox run <command>  - Выполнить команду в sandbox")
	fmt.Println("  qwen-claw sandbox list           - Список sandbox")
	fmt.Println("  qwen-claw sandbox clean          - Очистить старые sandbox")
	
	return nil
}

// sandboxRun выполняет команду в sandbox
func sandboxRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command is required")
	}
	
	command := strings.Join(args, " ")
	
	// Проверяем Docker
	if !isDockerAvailable() {
		return fmt.Errorf("docker is not available")
	}
	
	fmt.Printf("🔒 Выполнение команды в sandbox: %s\n\n", command)
	
	// Создаём менеджер sandbox
	sbConfig := sandbox.DefaultConfig()
	sbManager, _ := sandbox.NewManager(sbConfig, "/tmp/qwen-claw-sandbox")
	
	// Создаём sandbox
	_, err := sbManager.CreateSandbox("cli")
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}
	
	// Выполняем команду
	ctx := context.Background()
	result, err := sbManager.Execute(ctx, "cli", command)
	if err != nil {
		if _, ok := err.(sandbox.ErrSecurityViolation); ok {
			return fmt.Errorf("🚫 Blocked by sandbox: %w", err)
		}
		return fmt.Errorf("execution failed: %w", err)
	}
	
	if result.Stdout != "" {
		fmt.Println(result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprintln(os.Stderr, result.Stderr)
	}
	
	fmt.Printf("\n⏱️  Duration: %v\n", result.Duration)
	if result.ExitCode != 0 {
		fmt.Printf("❌ Exit code: %d\n", result.ExitCode)
	}
	
	return nil
}

// sandboxList показывает список sandbox
func sandboxList(cmd *cobra.Command, args []string) error {
	sbConfig := sandbox.DefaultConfig()
	sbManager, _ := sandbox.NewManager(sbConfig, "/tmp/qwen-claw-sandbox")
	
	sandboxes := sbManager.ListSandboxes()
	
	if len(sandboxes) == 0 {
		fmt.Println("📭 Нет активных sandbox")
		return nil
	}
	
	fmt.Printf("🔒 Active Sandboxes (%d):\n\n", len(sandboxes))
	for _, sb := range sandboxes {
		stats := sb.GetStats()
		fmt.Printf("📦 %s\n", stats["id"])
		fmt.Printf("   Status: %v\n", stats["running"])
		fmt.Printf("   Created: %s\n", stats["created"])
		fmt.Printf("   Last Used: %s\n", stats["last_used"])
		fmt.Printf("   Executions: %v\n", stats["exec_count"])
		fmt.Println()
	}
	
	return nil
}

// sandboxClean очищает старые sandbox
func sandboxClean(cmd *cobra.Command, args []string) error {
	sbConfig := sandbox.DefaultConfig()
	sbManager, _ := sandbox.NewManager(sbConfig, "/tmp/qwen-claw-sandbox")
	
	maxAge := 24 * time.Hour
	if len(args) > 0 {
		var hours int
		fmt.Sscanf(args[0], "%d", &hours)
		maxAge = time.Duration(hours) * time.Hour
	}
	
	fmt.Printf("🧹 Cleaning sandboxes older than %v...\n", maxAge)
	
	if err := sbManager.Cleanup(maxAge); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}
	
	fmt.Println("✅ Cleanup completed")
	return nil
}

// isDockerAvailable проверяет доступность Docker
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// degradationReport показывает отчёт о деградации
func degradationReport(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	memoryManager := memory.NewManager(cfg.MemoryDir)
	if err := memoryManager.Init(); err != nil {
		return fmt.Errorf("failed to init memory: %w", err)
	}

	agentInstance := agent.NewAgent(
		agent.AgentConfig{},
		memoryManager,
	)

	report := agentInstance.GetAntiDegradationSystem().GetDegradationReport()

	fmt.Printf("📊 Anti-Degradation Report\n")
	fmt.Printf("=========================\n\n")
	fmt.Printf("✅ Total tasks: %d\n", report.TotalTasks)
	fmt.Printf("⚠️  Degradations: %d\n", report.DegradationCount)
	fmt.Printf("📚 Lessons learned: %d\n", report.LessonsCount)
	fmt.Printf("📈 Efficiency: %.1f%%\n\n", report.Efficiency)

	if len(report.Patterns) > 0 {
		fmt.Println("Patterns detected:")
		for pattern, count := range report.Patterns {
			fmt.Printf("  - %s: %d\n", pattern, count)
		}
	}

	return nil
}

// degradationClear очищает историю деградации
func degradationClear(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Очищаем файлы
	dataDir := filepath.Join(os.Getenv("HOME"), "qwen-claw", ".qwen", "anti-degradation")
	os.Remove(filepath.Join(dataDir, "metrics.json"))
	os.Remove(filepath.Join(dataDir, "lessons.json"))

	fmt.Println("✅ Anti-degradation history cleared")
	return nil
}

