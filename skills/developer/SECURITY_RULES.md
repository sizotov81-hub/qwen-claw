# 🛡️ Правила безопасности разработчика qwen-claw

## ✅ **ДЕЛАЙТЕ:**

### 1. Хранение секретов
- Продолжайте хранить секреты **только** в `.env`
- Используйте переменные окружения для чувствительных данных
- Пример правильного `.env`:
```bash
QWEN_CLAW_TELEGRAM_TOKEN="your-token-here"
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123,456,789"
QWEN_CLAW_MODEL="gpt-4"
QWEN_CLAW_APPROVAL_MODE="auto-edit"
```

### 2. Контроль доступа
- Используйте `QWEN_CLAW_TELEGRAM_ALLOWED_USERS` для ограничения доступа
- Перечисляйте ID пользователей через запятую
- Проверяйте права доступа в начале обработки запроса

### 3. Git безопасность
- Проверяйте `.gitignore` перед каждым коммитом
- Убедитесь что `.env`, `*.db`, `*.log` не попали в коммит
- Используйте `git status` перед `git commit`

### 4. Подтверждения
- Используйте режим `auto-edit` для подтверждения изменений
- Для опасных операций используйте `plan` режим
- Никогда не используйте `yolo` в продакшене

---

## ❌ **НЕ ДЕЛАЙТЕ:**

### 1. Никогда не коммитьте `.env`
```bash
# ❌ ПЛОХО
git add .env
git commit -m "add config"

# ✅ ХОРОШО
echo ".env" >> .gitignore
git add .gitignore
```

### 2. Не передавайте токены в аргументах
```bash
# ❌ ПЛОХО
./qwen-claw --token "abc123"
./qwen-claw telegram "abc123"

# ✅ ХОРОШО
export QWEN_CLAW_TELEGRAM_TOKEN="abc123"
./qwen-claw telegram
```

### 3. Не отключайте подтверждение для опасных команд
```bash
# ❌ ПЛОХО
QWEN_CLAW_APPROVAL_MODE="yolo" rm -rf /tmp/*

# ✅ ХОРОШО
QWEN_CLAW_APPROVAL_MODE="auto-edit" rm -rf /tmp/*
# → Запросит подтверждение
```

### 4. Не логируйте секреты
```go
// ❌ ПЛОХО
logger.Infof("Token: %s", config.Token)
fmt.Println("Password:", password)

// ✅ ХОРОШО
logger.Infof("Token: %s", "***REDACTED***")
logger.Infof("Authenticated: %v", authenticated)
```

---

## 🔍 Чеклист перед коммитом

```bash
# 1. Проверка файлов
git status

# 2. Проверка .env
grep -r "\.env" .gitignore || echo ".env not in gitignore!"

# 3. Проверка на секреты
git diff --cached | grep -i "token\|secret\|password" && echo "⚠️ Возможные секреты!"

# 4. Запуск тестов
go test ./...

# 5. Линтер
golangci-lint run
```

---

## 📋 Примеры безопасного кода

### ✅ Правильно: Чтение конфига
```go
cfg, err := config.Load()
if err != nil {
    return fmt.Errorf("failed to load config")
}
// Token используется но не логируется
bot, err := telegram.NewBot(cfg.Telegram.Token)
```

### ✅ Правильно: Проверка доступа
```go
func (b *Bot) checkAccess(userID int64) bool {
    if len(b.config.AllowedUsers) == 0 {
        return true // Открытый доступ
    }
    for _, id := range b.config.AllowedUsers {
        if id == userID {
            return true
        }
    }
    return false
}
```

### ✅ Правильно: Безопасное логирование
```go
func logAuth(userID int64, success bool) {
    if success {
        logger.Infof("User %d authenticated", userID)
    } else {
        logger.Warnf("Auth failed for user %d", userID)
    }
    // Token/secret не логируются
}
```

---

## 🚨 Нарушения безопасности

Если обнаружили нарушение:

1. **Немедленно** отзовите скомпрометированные токены
2. Обновите `.env` с новыми секретами
3. Проверьте git history на утечки
4. Добавьте правило в `.gitignore`
5. Задокументируйте инцидент

---

**Помните:** Безопасность > Удобство

*Версия: 1.0.0 | Последнее обновление: 2026-03-22*
