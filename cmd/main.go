package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/qwen-claw/internal/agent"
	"github.com/user/qwen-claw/internal/config"
	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/skills"
	"github.com/user/qwen-claw/internal/scheduler"
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

	fmt.Println(response)
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

	fmt.Println(response)
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
				fmt.Println("History cleared")

			case "/memory":
				entries := memoryManager.ListEntries()
				if len(entries) == 0 {
					fmt.Println("Memory is empty")
				} else {
					for _, e := range entries {
						fmt.Printf("[%s] %s\n", e.Type, e.Content)
					}
				}

			case "/context":
				ctx := agentInstance.GetMemoryContext()
				if ctx == "" {
					fmt.Println("No context")
				} else {
					fmt.Println(ctx)
				}

			case "/exit", "/quit":
				fmt.Println("Goodbye!")
				return nil

			default:
				fmt.Printf("Unknown command: %s\n", input)
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

		fmt.Println("\n💬 " + response)
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
	fmt.Println("================\n")

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

func runTelegramBot(cmd *cobra.Command, args []string) error {
	log.Println("Loading configuration...")
	
	// Загружаем конфигурацию
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// НЕ логируем токен и чувствительные данные!
	log.Printf("Telegram enabled: %v", cfg.Telegram.Enabled)
	if len(cfg.Telegram.AllowedUsers) > 0 {
		log.Printf("Access restricted to %d user(s)", len(cfg.Telegram.AllowedUsers))
	} else {
		log.Println("Access: open for all users")
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
		log.Printf("Warning: Qwen Code CLI not found at %s", agentInstance.GetQwenPath())
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
		log.Printf("Access restricted to users: %v", cfg.Telegram.AllowedUsers)
	} else {
		log.Println("Access: open for all users")
	}

	// Показываем информацию о боте
	botInfo, err := bot.GetBotInfo()
	if err != nil {
		log.Printf("Warning: failed to get bot info: %v", err)
	} else {
		log.Printf("Bot username: %s", botInfo)
	}

	log.Println("Starting Telegram bot...")
	log.Println("Press Ctrl+C to stop")

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

	skillList := skillEngine.List()
	if len(skillList) == 0 {
		fmt.Println("No skills found")
		return nil
	}

	fmt.Printf("📦 Skills (%d total):\n\n", len(skillList))
	for _, s := range skillList {
		status := "✅"
		if !s.Enabled {
			status = "❌"
		}
		typeStr := string(s.Type)
		if typeStr == "" {
			typeStr = "unknown"
		}
		fmt.Printf("%s [%s] %s\n", status, typeStr, s.Name)
		fmt.Printf("   Description: %s\n", s.Description)
		fmt.Printf("   Commands: %s\n", strings.Join(s.Commands, ", "))
		fmt.Println()
	}

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

	skillPath := filepath.Join(cfg.SkillsDir, skillName, "skill.json")
	_, err = os.ReadFile(skillPath)
	if err != nil {
		return fmt.Errorf("skill '%s' not found: %w", skillName, err)
	}

	// Включаем навык (пока просто сообщаем, что нужно отредактировать файл)
	fmt.Printf("To enable skill '%s', edit %s and set 'enabled': true\n", skillName, skillPath)
	fmt.Println("Note: Skills are enabled by default when loaded from skill.json")
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

	skillPath := filepath.Join(cfg.SkillsDir, skillName, "skill.json")
	_, err = os.ReadFile(skillPath)
	if err != nil {
		return fmt.Errorf("skill '%s' not found: %w", skillName, err)
	}

	fmt.Printf("To disable skill '%s', edit %s and set 'enabled': false\n", skillName, skillPath)
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

	log.Printf("🔒 Secure Web UI starting at http://%s:%d", host, port)
	log.Printf("📝 Use secret phrase for authentication (see .web-secrets.json)")
	log.Println("Press Ctrl+C to stop")

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

	// Создаём и запускаем веб-сервер
	webServer := web.NewServer(
		*webConfig,
		agentInstance,
		memoryManager,
		sched,
	)

	return webServer.Start()
}

