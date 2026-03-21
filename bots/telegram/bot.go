package telegram

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/user/qwen-claw/internal/agent"
	"github.com/user/qwen-claw/internal/logger"
	"github.com/user/qwen-claw/internal/memory"
	"github.com/user/qwen-claw/internal/scheduler"
	"github.com/user/qwen-claw/internal/selfimprovement"
	"github.com/user/qwen-claw/internal/voice"
)

// BotConfig конфигурация бота
type BotConfig struct {
	// Token токен бота от BotFather
	Token string `json:"token"`

	// AllowedUsers разрешённые пользователи (пусто = все)
	AllowedUsers []int64 `json:"allowed_users"`

	// Timeout таймаут запросов
	Timeout time.Duration `json:"timeout"`
}

// Bot Telegram бот
type Bot struct {
	// config конфигурация
	config BotConfig

	// api Telegram API
	api *tgbotapi.BotAPI

	// agent AI агент (оболочка для Qwen Code CLI)
	agent *agent.Agent

	// memory менеджер памяти
	memory *memory.Manager

	// scheduler планировщик задач
	scheduler *scheduler.Scheduler
	
	// selfImprovement система самосовершенствования
	selfImprovement *selfimprovement.Engine
	
	// running флаг работы
	running bool
}

// NewBot создаёт нового бота
func NewBot(
	config BotConfig,
	agentInstance *agent.Agent,
	memoryManager *memory.Manager,
	schedulerInstance *scheduler.Scheduler,
) (*Bot, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("telegram token is required")
	}

	// Создаём HTTP клиент с поддержкой прокси
	client, err := createHTTPClient()
	if err != nil {
		logger.Infof("Warning: failed to create HTTP client: %v", err)
	}

	api, err := tgbotapi.NewBotAPIWithClient(config.Token, tgbotapi.APIEndpoint, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot api: %w", err)
	}
	
	// Создаём систему самосовершенствования
	projectRoot := os.Getenv("HOME") + "/qwen-claw"
	selfImprovement := selfimprovement.NewEngine(projectRoot)

	return &Bot{
		config:          config,
		api:             api,
		agent:           agentInstance,
		memory:          memoryManager,
		scheduler:       schedulerInstance,
		selfImprovement: selfImprovement,
	}, nil
}

// createHTTPClient создаёт HTTP клиент с прокси
func createHTTPClient() (*http.Client, error) {
	// Проверяем переменные окружения для прокси
	proxyURL := os.Getenv("HTTPS_PROXY")
	if proxyURL == "" {
		proxyURL = os.Getenv("HTTP_PROXY")
	}
	if proxyURL == "" {
		proxyURL = os.Getenv("TELEGRAM_PROXY")
	}

	// Создаём транспорт
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		Proxy: http.ProxyFromEnvironment,
	}

	// Если указан прокси, используем его
	if proxyURL != "" {
		logger.Infof("Using proxy: %s", proxyURL)
		proxyParsed, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyParsed)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}, nil
}

// Start запускает бота
func (b *Bot) Start() error {
	// Получаем информацию о боте
	u, err := b.api.GetMe()
	if err != nil {
		return fmt.Errorf("failed to get bot info: %w", err)
	}

	logger.Infof("Telegram bot started: @%s", u.UserName)

	b.running = true

	// Настраиваем обновления
	uConfig := tgbotapi.NewUpdate(0)
	uConfig.Timeout = 60

	// Создаём канал для обновлений
	updates := b.api.GetUpdatesChan(uConfig)

	// Обрабатываем обновления
	for update := range updates {
		if update.Message != nil {
			go b.handleMessage(update.Message)
		}
		if update.CallbackQuery != nil {
			go b.handleCallbackQuery(update.CallbackQuery)
		}
	}

	return nil
}

// Stop останавливает бота
func (b *Bot) Stop() {
	b.running = false
	b.api.StopReceivingUpdates()
	logger.Info("Telegram bot stopped")
}

// handleMessage обрабатывает сообщение
func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	// Игнорируем сообщения от других ботов
	if msg.From.IsBot {
		return
	}

	logger.Infof("Message from @%s (ID: %d): %s", msg.From.UserName, msg.From.ID, msg.Text)

	// Проверяем права доступа
	if len(b.config.AllowedUsers) > 0 {
		allowed := false
		for _, id := range b.config.AllowedUsers {
			if id == msg.From.ID {
				allowed = true
				break
			}
		}
		if !allowed {
			logger.Infof("❌ Access denied for user %d (@%s)", msg.From.ID, msg.From.UserName)
			b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ У вас нет доступа к этому боту.\nВаш ID: %d", msg.From.ID))
			return
		}
		logger.Infof("✅ Access granted for user %d (@%s)", msg.From.ID, msg.From.UserName)
	}

	// Обрабатываем команды
	if msg.IsCommand() {
		b.handleCommand(msg)
		return
	}

	// Обрабатываем документы
	if msg.Document != nil {
		b.handleDocument(msg)
		return
	}

	// Обрабатываем голосовые сообщения
	if msg.Voice != nil {
		b.handleVoice(msg)
		return
	}

	// Обычное сообщение - отправляем в Qwen
	b.handleQuery(msg)
}

// handleCommand обрабатывает команды
func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	command := strings.ToLower(msg.Command())

	switch command {
	case "start":
		response := fmt.Sprintf(
			"👋 Привет, %s!\n\n"+
				"Я **Qwen-Claw Bot** - AI ассистент на базе Qwen Code CLI.\n\n"+
				"📋 **Возможности**:\n"+
				"• Выполнение shell команд\n"+
				"• Работа с файлами\n"+
				"• Поиск по коду\n"+
				"• Запоминание фактов\n"+
				"• Git операции\n"+
				"• HTTP запросы\n"+
				"• Задачи по расписанию\n\n"+
				"Нажми на кнопку ниже для помощи.",
			msg.From.FirstName,
		)
		b.sendMessageWithKeyboard(msg.Chat.ID, response, tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🛠️ Навыки", "help_skills"),
				tgbotapi.NewInlineKeyboardButtonData("📋 Задачи", "help_tasks"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🧠 Память", "help_memory"),
			),
		))

	case "help":
		response := "📖 **Справка**\n\n" +
			"**Команды**:\n" +
			"/start - Запустить бота\n" +
			"/help - Эта справка\n" +
			"/status - Статус бота\n" +
			"/memory - Показать память\n" +
			"/clear - Очистить историю\n" +
			"/model - Текущая модель\n" +
			"/tasks - Список задач\n" +
			"/addtask - Добавить задачу\n\n" +
			"**Примеры запросов**:\n" +
			"• `Создай файл test.go`\n" +
			"• `Выполни pwd`\n" +
			"• `Найди все .go файлы`\n" +
			"• `Запомни: проект на Go 1.21`"
		b.sendMessage(msg.Chat.ID, response)

	case "status":
		qwenPath := b.agent.GetQwenPath()
		qwenAvailable := b.agent.CheckQwenAvailable()

		status := "✅ Доступен"
		if !qwenAvailable {
			status = "❌ Не доступен"
		}

		response := fmt.Sprintf(
			"📊 **Статус**\n\n"+
				"Бот: %s\n"+
				"Qwen CLI: %s (%s)\n"+
				"Память: %d записей",
			b.runningStr(),
			status,
			qwenPath,
			len(b.memory.ListEntries()),
		)
		b.sendMessage(msg.Chat.ID, response)

	case "memory":
		entries := b.memory.ListEntries()
		if len(entries) == 0 {
			b.sendMessage(msg.Chat.ID, "📭 Память пуста")
			return
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("📚 **Память** (%d записей):\n\n", len(entries)))

		maxEntries := 10
		count := 0
		for _, e := range entries {
			if count >= maxEntries {
				response.WriteString(fmt.Sprintf("\n_... и ещё %d записей_", len(entries)-maxEntries))
				break
			}
			response.WriteString(fmt.Sprintf("• [%s] %s\n", e.Type, e.Content))
			count++
		}

		b.sendMessage(msg.Chat.ID, response.String())

	case "clear":
		if err := b.memory.Clear(); err != nil {
			b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
			return
		}
		b.sendMessage(msg.Chat.ID, "🗑️ Память очищена")

	case "model":
		model := b.agent.GetModel()
		if model == "" {
			model = "default (из Qwen CLI)"
		}
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("🤖 **Модель**: %s", model))

	case "skills":
		skills := b.agent.GetSkillEngine().List()
		if len(skills) == 0 {
			b.sendMessage(msg.Chat.ID, "🛠️ Нет навыков")
			return
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("🛠️ **Навыки** (%d):\n\n", len(skills)))
		for _, s := range skills {
			status := "✅"
			if !s.Enabled {
				status = "❌"
			}
			response.WriteString(fmt.Sprintf("%s *%s* - %s\n", status, s.Name, s.Description))
		}
		b.sendMessage(msg.Chat.ID, response.String())

	case "tasks":
		tasks := b.scheduler.ListTasks()
		if len(tasks) == 0 {
			b.sendMessage(msg.Chat.ID, "📋 Нет запланированных задач")
			return
		}

		var response strings.Builder
		response.WriteString(fmt.Sprintf("📋 **Задачи** (%d):\n\n", len(tasks)))
		for _, t := range tasks {
			status := "✅"
			if !t.Enabled {
				status = "❌"
			}
			response.WriteString(fmt.Sprintf("%s *%s*\n", status, t.Name))
			response.WriteString(fmt.Sprintf("  Schedule: `%s`\n", t.Schedule))
			if t.NextRun != nil {
				response.WriteString(fmt.Sprintf("  Next: %s\n", t.NextRun.Format("2006-01-02 15:04")))
			}
		}
		b.sendMessage(msg.Chat.ID, response.String())

	case "addtask":
		// Парсим аргументы: /addtask <name> <schedule> <command>
		args := strings.Fields(msg.CommandArguments())
		if len(args) < 3 {
			response := "📝 **Добавление задачи**\n\n" +
				"Используй формат:\n" +
				"`/addtask <name> <schedule> <command>`\n\n" +
				"**Примеры:**\n" +
				"`/addtask backup @daily Сделай резервную копию проекта`\n" +
				"`/addtask report '0 9 * * 1' Создай отчёт за неделю`\n\n" +
				"**Расписание:**\n" +
				"`@hourly` - каждый час\n" +
				"`@daily` - каждый день\n" +
				"`@weekly` - каждую неделю\n" +
				"`0 9 * * *` - в 9:00 ежедневно"
			b.sendMessage(msg.Chat.ID, response)
			return
		}

		name := args[0]
		schedule := args[1]
		command := strings.Join(args[2:], " ")

		task, err := b.scheduler.AddTask(name, name, command, schedule)
		if err != nil {
			b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка: %v", err))
			return
		}

		response := fmt.Sprintf(
			"✅ **Задача добавлена**\n\n"+
				"**ID:** `%s`\n"+
				"**Имя:** %s\n"+
				"**Расписание:** `%s`\n"+
				"**Команда:** %s\n",
			task.ID, task.Name, task.Schedule, task.Command,
		)
		if task.NextRun != nil {
			response += fmt.Sprintf("**След. запуск:** %s", task.NextRun.Format("2006-01-02 15:04"))
		}
		b.sendMessage(msg.Chat.ID, response)

	default:
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❓ Неизвестная команда: /%s\nИспользуй /help", command))
	}
}

// handleQuery обрабатывает обычный запрос
func (b *Bot) handleQuery(msg *tgbotapi.Message) {
	// Показываем индикатор "печатает"
	b.sendChatAction(msg.Chat.ID, "typing")

	// Отправляем промежуточное сообщение "Думаю..."
	thinkingMsg := b.sendMessage(msg.Chat.ID, "🤔 **Думаю...**")

	query := msg.Text

	// Сохраняем в память
	b.memory.AddMessage("user", query)

	// Отправляем запрос в Qwen
	ctx := context.Background()
	if b.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.config.Timeout)
		defer cancel()
	}

	response, err := b.agent.Run(ctx, query)

	// Сохраняем ответ в память
	if err == nil {
		b.memory.AddMessage("assistant", response)
	}

	// Отправляем ответ
	if err != nil {
		b.deleteMessage(msg.Chat.ID, thinkingMsg)
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ **Ошибка**: %v", err))
		return
	}

	// Форматируем ответ (Markdown)
	response = b.formatResponse(response)

	// Telegram имеет лимит 4096 символов на сообщение
	if len(response) > 4000 {
		// Разбиваем на части
		parts := b.splitMessage(response, 4000)
		for i, part := range parts {
			if i == len(parts)-1 {
				b.sendMessage(msg.Chat.ID, part)
			} else {
				b.sendMessage(msg.Chat.ID, part)
				time.Sleep(100 * time.Millisecond)
			}
		}
	} else {
		b.sendMessage(msg.Chat.ID, response)
	}

	// Удаляем сообщение "Думаю..."
	b.deleteMessage(msg.Chat.ID, thinkingMsg)
}

// handleDocument обрабатывает загруженные документы
func (b *Bot) handleDocument(msg *tgbotapi.Message) {
	b.sendChatAction(msg.Chat.ID, "typing")

	doc := msg.Document
	fileName := doc.FileName
	fileSize := doc.FileSize

	// Получаем файл
	fileURL, err := b.api.GetFileDirectURL(doc.FileID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка загрузки файла: %v", err))
		return
	}

	response := fmt.Sprintf(
		"📎 **Файл получен**\n\n"+
			"**Имя:** `%s`\n"+
			"**Размер:** %d bytes\n"+
			"**URL:** %s\n\n"+
			"Что сделать с этим файлом?",
		fileName, fileSize, fileURL,
	)
	b.sendMessage(msg.Chat.ID, response)
}

// handleVoice обрабатывает голосовые сообщения
func (b *Bot) handleVoice(msg *tgbotapi.Message) {
	b.sendChatAction(msg.Chat.ID, "typing")

	voice := msg.Voice
	
	// Получаем файл
	file := tgbotapi.FileConfig{
		FileID: voice.FileID,
	}
	
	fileURL, err := b.api.GetFileDirectURL(voice.FileID)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка загрузки голосового: %v", err))
		return
	}
	
	// Скачиваем
	resp, err := http.Get(fileURL)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка скачивания: %v", err))
		return
	}
	defer resp.Body.Close()
	
	voiceData, err := io.ReadAll(resp.Body)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка чтения: %v", err))
		return
	}
	
	// Пробуем распознать через whisper
	whisper := voice.NewWhisper()
	
	if !whisper.IsInstalled() {
		// Whisper не установлен - предлагаем установить
		response := "🎤 **Голосовое сообщение получено**\n\n" +
			"Для распознавания нужно установить whisper.cpp.\n\n" +
			"**Что будет установлено:**\n" +
			"- whisper.cpp (распознавание речи)\n" +
			"- ffmpeg (конвертация аудио)\n" +
			"- Модель base (~100MB)\n\n" +
			"**Команды:**\n" +
			"```\n" + strings.Join(voice.GetInstallationCommands(), "\n") + "\n```\n\n" +
			"Установить? Напишите `ПОДТВЕРЖДАЮ установку whisper`"
		
		b.sendMessage(msg.Chat.ID, response)
		return
	}
	
	// Распознаём
	text, err := whisper.Transcribe(voiceData)
	if err != nil {
		b.sendMessage(msg.Chat.ID, fmt.Sprintf("❌ Ошибка распознавания: %v", err))
		return
	}
	
	// Отправляем распознанный текст
	response := fmt.Sprintf(
		"🎤 **Голосовое распознано**\n\n"+
			"**Текст:**\n_%s_\n\n"+
			"Продолжаю диалог...",
		text,
	)
	b.sendMessage(msg.Chat.ID, response)
	
	// Отправляем распознанный текст в агент для обработки
	b.handleQuery(&tgbotapi.Message{
		Chat: msg.Chat,
		Text: text,
		From: msg.From,
	})
}

// IsInstalled проверяет установку whisper
func (w *Whisper) IsInstalled() bool {
	if _, err := os.Stat(w.whisperPath); err != nil {
		return false
	}
	if _, err := os.Stat(w.modelPath); err != nil {
		return false
	}
	return true
}

// sendMessage отправляет сообщение
func (b *Bot) sendMessage(chatID int64, text string) int {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.DisableWebPagePreview = true

	result, err := b.api.Send(msg)
	if err != nil {
		logger.Infof("Failed to send message: %v", err)
		return 0
	}
	return result.MessageID
}

// deleteMessage удаляет сообщение
func (b *Bot) deleteMessage(chatID int64, messageID int) {
	if messageID > 0 {
		_, _ = b.api.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
	}
}

// sendChatAction отправляет действие (typing, etc.)
func (b *Bot) sendChatAction(chatID int64, action string) {
	actionMsg := tgbotapi.NewChatAction(chatID, action)
	_, _ = b.api.Request(actionMsg)
}

// formatResponse форматирует ответ для Telegram
func (b *Bot) formatResponse(response string) string {
	// Экранируем специальные символы Markdown
	response = strings.ReplaceAll(response, "_", "\\_")
	response = strings.ReplaceAll(response, "*", "\\*")
	response = strings.ReplaceAll(response, "[", "\\[")
	response = strings.ReplaceAll(response, "`", "\\`")

	// Заменяем блоки кода на формат Telegram
	response = strings.ReplaceAll(response, "```go", "```go\n")
	response = strings.ReplaceAll(response, "```python", "```python\n")
	response = strings.ReplaceAll(response, "```bash", "```bash\n")
	response = strings.ReplaceAll(response, "```javascript", "```javascript\n")
	response = strings.ReplaceAll(response, "```", "```\n")

	return response
}

// splitMessage разбивает длинное сообщение на части
func (b *Bot) splitMessage(text string, maxLen int) []string {
	var parts []string

	for len(text) > maxLen {
		// Ищем последнюю новую строку в пределах лимита
		cutIndex := maxLen
		for i := maxLen; i > 0; i-- {
			if text[i] == '\n' {
				cutIndex = i
				break
			}
		}

		parts = append(parts, text[:cutIndex])
		text = text[cutIndex+1:]
	}

	if len(text) > 0 {
		parts = append(parts, text)
	}

	return parts
}

// handleCallbackQuery обрабатывает нажатия на inline-кнопки
func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	logger.Infof("Callback query from %d: %s", callback.From.ID, callback.Data)

	data := callback.Data
	chatID := callback.Message.Chat.ID

	switch data {
	case "help_skills":
		response := "🛠️ **Навыки**\n\n" +
			"Qwen-Claw поддерживает навыки:\n\n" +
			"• 🐚 **shell** - shell команды\n" +
			"• 📁 **file** - работа с файлами\n" +
			"• 🔍 **search** - поиск по коду\n" +
			"• 🧠 **memory** - управление памятью\n" +
			"• 🌲 **git** - Git операции\n" +
			"• 🌐 **http** - HTTP запросы\n" +
			"• 🔔 **notify** - уведомления\n\n" +
			"Используй: `/skills` для списка"
		b.sendEditMessage(chatID, callback.Message.MessageID, response)

	case "help_tasks":
		response := "📋 **Задачи**\n\n" +
			"Планировщик позволяет выполнять задачи по расписанию.\n\n" +
			"**Расписание:**\n" +
			"• `@hourly` - каждый час\n" +
			"• `@daily` - ежедневно\n" +
			"• `@weekly` - еженедельно\n" +
			"• `0 9 * * *` - в 9:00\n\n" +
			"Используй: `/tasks` для списка"
		b.sendEditMessage(chatID, callback.Message.MessageID, response)

	case "help_memory":
		response := "🧠 **Память**\n\n" +
			"Qwen-Claw запоминает:\n" +
			"• Ваши запросы\n" +
			"• Ответы ассистента\n" +
			"• Важные факты\n\n" +
			"Команды:\n" +
			"• `/memory` - показать\n" +
			"• `/clear` - очистить"
		b.sendEditMessage(chatID, callback.Message.MessageID, response)

	case "main_menu":
		b.sendMainMenu(chatID)
	}

	// Отвечаем на callback
	answer := tgbotapi.NewCallback(callback.ID, "")
	_, _ = b.api.Request(answer)
}

// sendEditMessage редактирует сообщение
func (b *Bot) sendEditMessage(chatID int64, messageID int, text string) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛠️ Навыки", "help_skills"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Задачи", "help_tasks"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 Память", "help_memory"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Меню", "main_menu"),
		),
	)
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = &keyboard
	_, _ = b.api.Send(msg)
}

// sendMainMenu отправляет главное меню
func (b *Bot) sendMainMenu(chatID int64) {
	response := "🏠 **Главное меню**\n\n" +
		"Выберите раздел помощи или отправьте мне запрос."
	b.sendMessageWithKeyboard(chatID, response, tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛠️ Навыки", "help_skills"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Задачи", "help_tasks"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🧠 Память", "help_memory"),
		),
	))
}

// sendMessageWithKeyboard отправляет сообщение с клавиатурой
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	_, _ = b.api.Send(msg)
}

// runningStr возвращает строковое представление статуса
func (b *Bot) runningStr() string {
	if b.running {
		return "🟢 Работает"
	}
	return "🔴 Остановлен"
}

// GetBotInfo возвращает информацию о боте
func (b *Bot) GetBotInfo() (string, error) {
	u, err := b.api.GetMe()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("@%s", u.UserName), nil
}
