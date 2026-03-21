package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SkillType тип навыка
type SkillType string

const (
	SkillTypeBuiltin  SkillType = "builtin"
	SkillTypeExternal SkillType = "external"
	SkillTypeScript   SkillType = "script"
)

// Skill определение навыка
type Skill struct {
	// Name имя навыка
	Name string `json:"name"`
	
	// Description описание навыка
	Description string `json:"description"`
	
	// Type тип навыка
	Type SkillType `json:"type"`
	
	// EntryPoint точка входа (для external/script)
	EntryPoint string `json:"entry_point,omitempty"`
	
	// Commands список команд, которые обрабатывает навык
	Commands []string `json:"commands"`
	
	// Enabled включён ли навык
	Enabled bool `json:"enabled"`
}

// SkillRequest запрос к навыку
type SkillRequest struct {
	// Command команда
	Command string `json:"command"`
	
	// Args аргументы
	Args []string `json:"args,omitempty"`
	
	// Input входные данные
	Input string `json:"input,omitempty"`
	
	// Context контекст выполнения
	Context map[string]interface{} `json:"context,omitempty"`
}

// SkillResult результат выполнения навыка
type SkillResult struct {
	// Success успешно ли выполнено
	Success bool `json:"success"`
	
	// Output выходные данные
	Output string `json:"output,omitempty"`
	
	// Error ошибка
	Error string `json:"error,omitempty"`
	
	// Data дополнительные данные
	Data interface{} `json:"data,omitempty"`
}

// Engine движок навыков
type Engine struct {
	// skillsDir директория с навыками
	skillsDir string

	// skills загруженные навыки
	skills map[string]*Skill

	// memoryManager менеджер памяти (опционально)
	memoryManager interface{}
}

// NewEngine создаёт новый движок навыков
func NewEngine(skillsDir string) *Engine {
	return &Engine{
		skillsDir: skillsDir,
		skills:    make(map[string]*Skill),
	}
}

// NewAgentSkillEngine создаёт движок навыков для агента (с загруженными builtin навыками)
func NewAgentSkillEngine() *Engine {
	e := &Engine{
		skillsDir: "",
		skills:    make(map[string]*Skill),
	}
	e.loadBuiltinSkills()
	return e
}

// SetMemoryManager устанавливает менеджер памяти для интеграции
func (e *Engine) SetMemoryManager(mm interface{}) {
	e.memoryManager = mm
}

// Load загружает все навыки из директории
func (e *Engine) Load() error {
	// Загружаем встроенные навыки
	e.loadBuiltinSkills()
	
	// Загружаем внешние навыки из директории
	return e.loadExternalSkills()
}

// loadBuiltinSkills загружает встроенные навыки
func (e *Engine) loadBuiltinSkills() {
	builtins := []Skill{
		{
			Name:        "shell",
			Description: "Выполнение shell команд",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"exec", "run", "shell"},
			Enabled:     true,
		},
		{
			Name:        "file",
			Description: "Операции с файлами",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"read", "write", "edit", "file"},
			Enabled:     true,
		},
		{
			Name:        "search",
			Description: "Поиск по файлам и коду",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"search", "grep", "find"},
			Enabled:     true,
		},
		{
			Name:        "memory",
			Description: "Управление памятью и контекстом",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"remember", "recall", "memory"},
			Enabled:     true,
		},
		{
			Name:        "git",
			Description: "Работа с Git репозиториями",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"git", "commit", "push", "pull", "status"},
			Enabled:     true,
		},
		{
			Name:        "http",
			Description: "HTTP запросы (GET, POST, etc.)",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"http", "curl", "get", "post", "request"},
			Enabled:     true,
		},
		{
			Name:        "notify",
			Description: "Отправка уведомлений",
			Type:        SkillTypeBuiltin,
			Commands:    []string{"notify", "notification", "alert", "message"},
			Enabled:     true,
		},
	}

	for i := range builtins {
		e.skills[builtins[i].Name] = &builtins[i]
	}
}

// loadExternalSkills загружает внешние навыки из директории
func (e *Engine) loadExternalSkills() error {
	entries, err := os.ReadDir(e.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		skillPath := filepath.Join(e.skillsDir, entry.Name())
		skill, err := e.loadSkillFromDir(skillPath)
		if err != nil {
			fmt.Printf("Warning: failed to load skill %s: %v\n", entry.Name(), err)
			continue
		}
		
		e.skills[skill.Name] = skill
	}
	
	return nil
}

// loadSkillFromDir загружает навык из директории
func (e *Engine) loadSkillFromDir(path string) (*Skill, error) {
	// Читаем skill.json
	skillFile := filepath.Join(path, "skill.json")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill.json: %w", err)
	}
	
	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("failed to parse skill.json: %w", err)
	}
	
	// Определяем тип и точку входа
	if skill.Type == "" {
		// Пытаемся определить тип по файлам
		if _, err := os.Stat(filepath.Join(path, "main.sh")); err == nil {
			skill.Type = SkillTypeScript
			skill.EntryPoint = "main.sh"
		} else if _, err := os.Stat(filepath.Join(path, "main.py")); err == nil {
			skill.Type = SkillTypeScript
			skill.EntryPoint = "main.py"
		} else if _, err := os.Stat(filepath.Join(path, "main.go")); err == nil {
			skill.Type = SkillTypeExternal
			skill.EntryPoint = "main"
		} else {
			return nil, fmt.Errorf("no entry point found for skill")
		}
	}
	
	skill.Enabled = true
	return &skill, nil
}

// Execute выполняет навык
func (e *Engine) Execute(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	skill := e.FindSkillByCommand(req.Command)
	if skill == nil {
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("no skill found for command: %s", req.Command),
		}, nil
	}
	
	if !skill.Enabled {
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("skill %s is disabled", skill.Name),
		}, nil
	}
	
	switch skill.Type {
	case SkillTypeBuiltin:
		return e.executeBuiltin(ctx, skill, req)
	case SkillTypeExternal, SkillTypeScript:
		return e.executeExternal(ctx, skill, req)
	default:
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("unknown skill type: %s", skill.Type),
		}, nil
	}
}

// FindSkillByCommand находит навык по команде
func (e *Engine) FindSkillByCommand(command string) *Skill {
	for _, skill := range e.skills {
		for _, cmd := range skill.Commands {
			if cmd == command {
				return skill
			}
		}
	}
	return nil
}

// executeBuiltin выполняет встроенный навык
func (e *Engine) executeBuiltin(ctx context.Context, skill *Skill, req *SkillRequest) (*SkillResult, error) {
	switch skill.Name {
	case "shell":
		return e.execShell(ctx, req)
	case "file":
		return e.execFile(ctx, req)
	case "search":
		return e.execSearch(ctx, req)
	case "memory":
		return e.execMemory(ctx, req)
	case "git":
		return e.execGit(ctx, req)
	case "http":
		return e.execHTTP(ctx, req)
	case "notify":
		return e.execNotify(ctx, req)
	default:
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("unknown builtin skill: %s", skill.Name),
		}, nil
	}
}

// execShell выполняет shell команду
func (e *Engine) execShell(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) == 0 {
		return &SkillResult{
			Success: false,
			Error:   "no command specified",
		}, nil
	}
	
	cmd := exec.CommandContext(ctx, "bash", "-c", strings.Join(req.Args, " "))
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &SkillResult{
				Success: false,
				Error:   fmt.Sprintf("exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
			}, nil
		}
		return &SkillResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	
	return &SkillResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// execFile выполняет операции с файлами
func (e *Engine) execFile(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) < 2 {
		return &SkillResult{
			Success: false,
			Error:   "usage: file <read|write> <path> [content]",
		}, nil
	}
	
	op := req.Args[0]
	path := req.Args[1]
	
	switch op {
	case "read":
		content, err := os.ReadFile(path)
		if err != nil {
			return &SkillResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
		return &SkillResult{
			Success: true,
			Output:  string(content),
		}, nil
		
	case "write":
		if len(req.Args) < 3 {
			return &SkillResult{
				Success: false,
				Error:   "no content provided for write",
			}, nil
		}
		content := strings.Join(req.Args[2:], " ")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return &SkillResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}
		return &SkillResult{
			Success: true,
			Output:  fmt.Sprintf("written %d bytes to %s", len(content), path),
		}, nil
		
	default:
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("unknown file operation: %s", op),
		}, nil
	}
}

// execSearch выполняет поиск
func (e *Engine) execSearch(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) == 0 {
		return &SkillResult{
			Success: false,
			Error:   "no search pattern specified",
		}, nil
	}
	
	pattern := req.Args[0]
	searchPath := "."
	if len(req.Args) > 1 {
		searchPath = req.Args[1]
	}
	
	cmd := exec.CommandContext(ctx, "grep", "-r", "--include=*.go", "--include=*.py", "--include=*.js", "--include=*.ts", pattern, searchPath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				// Ничего не найдено
				return &SkillResult{
					Success: true,
					Output:  "nothing found",
				}, nil
			}
			return &SkillResult{
				Success: false,
				Error:   fmt.Sprintf("search failed: %s", string(exitErr.Stderr)),
			}, nil
		}
		return &SkillResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	
	return &SkillResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// execMemory управляет памятью
func (e *Engine) execMemory(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) == 0 {
		return &SkillResult{
			Success: false,
			Error:   "usage: memory <add|search|list|clear> [args]",
		}, nil
	}

	op := req.Args[0]

	switch op {
	case "add", "remember":
		if len(req.Args) < 2 {
			return &SkillResult{
				Success: false,
				Error:   "no content to remember",
			}, nil
		}
		content := strings.Join(req.Args[1:], " ")
		// Если memoryManager подключён, используем его
		if e.memoryManager != nil {
			if mm, ok := e.memoryManager.(interface{ Remember(string, string, map[string]interface{}) (*interface{}, error) }); ok {
				_, _ = mm.Remember("fact", content, nil)
				return &SkillResult{
					Success: true,
					Output:  fmt.Sprintf("Remembered: %s", content),
				}, nil
			}
		}
		return &SkillResult{
			Success: true,
			Output:  fmt.Sprintf("Would remember: %s", content),
		}, nil

	case "search", "recall":
		if len(req.Args) < 2 {
			return &SkillResult{
				Success: false,
				Error:   "no search query",
			}, nil
		}
		query := strings.Join(req.Args[1:], " ")
		if e.memoryManager != nil {
			if mm, ok := e.memoryManager.(interface{ Recall(string, int) []*interface{} }); ok {
				results := mm.Recall(query, 10)
				if len(results) == 0 {
					return &SkillResult{
						Success: true,
						Output:  "Nothing found in memory",
					}, nil
				}
				return &SkillResult{
					Success: true,
					Output:  fmt.Sprintf("Found %d entries", len(results)),
				}, nil
			}
		}
		return &SkillResult{
			Success: true,
			Output:  fmt.Sprintf("Would search for: %s", query),
		}, nil

	case "list":
		if e.memoryManager != nil {
			if mm, ok := e.memoryManager.(interface{ ListEntries() []*interface{} }); ok {
				results := mm.ListEntries()
				if len(results) == 0 {
					return &SkillResult{
						Success: true,
						Output:  "Memory is empty",
					}, nil
				}
				return &SkillResult{
					Success: true,
					Output:  fmt.Sprintf("%d entries in memory", len(results)),
				}, nil
			}
		}
		return &SkillResult{
			Success: true,
			Output:  "Memory manager not connected",
		}, nil

	case "clear":
		if e.memoryManager != nil {
			if mm, ok := e.memoryManager.(interface{ Clear() error }); ok {
				if err := mm.Clear(); err != nil {
					return &SkillResult{
						Success: false,
						Error:   err.Error(),
					}, nil
				}
				return &SkillResult{
					Success: true,
					Output:  "Memory cleared",
				}, nil
			}
		}
		return &SkillResult{
			Success: true,
			Output:  "Would clear memory",
		}, nil

	default:
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("unknown memory operation: %s", op),
		}, nil
	}
}

// execGit выполняет Git команды
func (e *Engine) execGit(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) == 0 {
		return &SkillResult{
			Success: false,
			Error:   "usage: git <command> [args...]\nExamples: git status, git commit -m 'message', git push",
		}, nil
	}

	// Проверяем, что git установлен
	if _, err := exec.LookPath("git"); err != nil {
		return &SkillResult{
			Success: false,
			Error:   "git not found in PATH. Please install git.",
		}, nil
	}

	cmd := exec.CommandContext(ctx, "git", req.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &SkillResult{
				Success: false,
				Error:   fmt.Sprintf("git exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
			}, nil
		}
		return &SkillResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &SkillResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// execHTTP выполняет HTTP запросы
func (e *Engine) execHTTP(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) < 2 {
		return &SkillResult{
			Success: false,
			Error:   "usage: http <method> <url> [body/headers...]\nExamples: http get https://api.example.com, http post https://api.example.com '{\"key\":\"value\"}'",
		}, nil
	}

	method := strings.ToUpper(req.Args[0])
	url := req.Args[1]

	// Проверяем, что curl установлен
	if _, err := exec.LookPath("curl"); err != nil {
		return &SkillResult{
			Success: false,
			Error:   "curl not found in PATH. Please install curl.",
		}, nil
	}

	// Формируем команду curl
	args := []string{"-s", "-X", method}

	// Добавляем заголовки и тело запроса
	if len(req.Args) > 2 {
		body := strings.Join(req.Args[2:], " ")
		// Если есть тело, добавляем как data
		if method == "POST" || method == "PUT" || method == "PATCH" {
			args = append(args, "-H", "Content-Type: application/json")
			args = append(args, "-d", body)
		} else {
			// Для GET добавляем как query params если нужно
			args = append(args, "-d", body)
		}
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "curl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &SkillResult{
				Success: false,
				Error:   fmt.Sprintf("curl exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
			}, nil
		}
		return &SkillResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &SkillResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// execNotify отправляет уведомления
func (e *Engine) execNotify(ctx context.Context, req *SkillRequest) (*SkillResult, error) {
	if len(req.Args) == 0 {
		return &SkillResult{
			Success: false,
			Error:   "usage: notify <message>\nExamples: notify 'Task completed', notify 'Build failed'",
		}, nil
	}

	message := strings.Join(req.Args, " ")
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Проверяем доступные способы отправки уведомлений
	notifyCmd := ""
	notifyArgs := []string{}

	// Проверяем notify-send (Linux desktop)
	if _, err := exec.LookPath("notify-send"); err == nil {
		notifyCmd = "notify-send"
		notifyArgs = []string{"Qwen-Claw", message}
	}

	// Проверяем osascript (macOS)
	if notifyCmd == "" {
		if _, err := exec.LookPath("osascript"); err == nil {
			notifyCmd = "osascript"
			notifyArgs = []string{"-e", fmt.Sprintf(`display notification "%s" with title "Qwen-Claw"`, message)}
		}
	}

	// Проверяем PowerShell (Windows)
	if notifyCmd == "" {
		if _, err := exec.LookPath("powershell"); err == nil {
			notifyCmd = "powershell"
			notifyArgs = []string{"-Command", fmt.Sprintf(`[System.Reflection.Assembly]::LoadWithPartialName("System.Windows.Forms"); [System.Windows.Forms.MessageBox]::Show("%s", "Qwen-Claw")`, message)}
		}
	}

	// Если ничего не найдено, просто логируем
	if notifyCmd == "" {
		logOutput := fmt.Sprintf("[%s] Notification: %s", timestamp, message)
		// Сохраняем в файл уведомлений
		homeDir, _ := os.UserHomeDir()
		notifyLog := filepath.Join(homeDir, ".qwen-claw", "notifications.log")
		os.MkdirAll(filepath.Dir(notifyLog), 0755)
		f, err := os.OpenFile(notifyLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString(logOutput + "\n")
			f.Close()
		}
		return &SkillResult{
			Success: true,
			Output:  fmt.Sprintf("Notification logged: %s\n(no desktop notification system found)", message),
		}, nil
	}

	// Отправляем уведомление
	cmd := exec.CommandContext(ctx, notifyCmd, notifyArgs...)
	if err := cmd.Run(); err != nil {
		return &SkillResult{
			Success: false,
			Error:   fmt.Sprintf("failed to send notification: %v", err),
		}, nil
	}

	return &SkillResult{
		Success: true,
		Output:  fmt.Sprintf("Notification sent: %s", message),
	}, nil
}

// executeExternal выполняет внешний навык
func (e *Engine) executeExternal(ctx context.Context, skill *Skill, req *SkillRequest) (*SkillResult, error) {
	skillPath := filepath.Join(e.skillsDir, skill.Name)
	entryPoint := skill.EntryPoint
	
	var cmd *exec.Cmd
	
	if skill.Type == SkillTypeScript {
		// Определяем интерпретатор по расширению
		switch filepath.Ext(entryPoint) {
		case ".sh":
			cmd = exec.CommandContext(ctx, "bash", filepath.Join(skillPath, entryPoint))
		case ".py":
			cmd = exec.CommandContext(ctx, "python3", filepath.Join(skillPath, entryPoint))
		default:
			return &SkillResult{
				Success: false,
				Error:   fmt.Sprintf("unsupported script extension: %s", entryPoint),
			}, nil
		}
	} else {
		// Compiled binary
		cmd = exec.CommandContext(ctx, filepath.Join(skillPath, entryPoint))
	}
	
	// Передаем аргументы и input
	cmd.Args = append(cmd.Args, req.Args...)
	if req.Input != "" {
		cmd.Stdin = strings.NewReader(req.Input)
	}
	
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &SkillResult{
				Success: false,
				Error:   fmt.Sprintf("exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
			}, nil
		}
		return &SkillResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	
	return &SkillResult{
		Success: true,
		Output:  string(output),
	}, nil
}

// List возвращает список всех навыков
func (e *Engine) List() []*Skill {
	result := make([]*Skill, 0, len(e.skills))
	for _, skill := range e.skills {
		result = append(result, skill)
	}
	return result
}

// Get возвращает навык по имени
func (e *Engine) Get(name string) *Skill {
	return e.skills[name]
}
