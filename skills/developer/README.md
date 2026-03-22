# 🛠️ Developer Skill для Qwen-Claw

## 📋 Описание

Системный навык разработчика для безопасной работы с кодом проекта qwen-claw.

## 🚀 Использование

```bash
# Показать справку
qwen-claw dev

# Показать правила безопасности
qwen-claw dev rules

# Проверить проект на нарушения
qwen-claw dev check

# Инициализировать проект
qwen-claw dev init
```

## 📜 Правила безопасности

### ✅ ДЕЛАЙТЕ:

1. **Храните секреты в `.env`**
   ```bash
   QWEN_CLAW_TELEGRAM_TOKEN="your-token"
   QWEN_CLAW_TELEGRAM_ALLOWED_USERS="123,456"
   ```

2. **Используйте `QWEN_CLAW_TELEGRAM_ALLOWED_USERS`**
   - Контролируйте доступ к боту
   - Перечисляйте ID через запятую

3. **Проверяйте `.gitignore` перед коммитом**
   ```bash
   git status
   git ls-files .env  # Должно быть пусто
   ```

4. **Используйте режим `auto-edit`**
   ```bash
   QWEN_CLAW_APPROVAL_MODE="auto-edit"
   ```

### ❌ НЕ ДЕЛАЙТЕ:

1. **Не коммитьте `.env`**
   ```bash
   # ❌
   git add .env
   
   # ✅
   echo ".env" >> .gitignore
   ```

2. **Не передавайте токены в аргументах**
   ```bash
   # ❌
   ./qwen-claw --token "abc123"
   
   # ✅
   export QWEN_CLAW_TELEGRAM_TOKEN="abc123"
   ```

3. **Не отключайте подтверждение для опасных команд**
   ```bash
   # ❌
   QWEN_CLAW_APPROVAL_MODE="yolo" rm -rf /
   
   # ✅
   QWEN_CLAW_APPROVAL_MODE="auto-edit" rm -rf /
   ```

4. **Не логируйте секреты**
   ```go
   // ❌
   logger.Info("Token: ", token)
   
   // ✅
   logger.Info("Token: ***REDACTED***")
   ```

## 🔍 Проверка безопасности

```bash
qwen-claw dev check
```

Проверяет:
- ✅ `.env` не в git
- ✅ `.gitignore` существует и содержит нужные записи
- ✅ Логи не содержат секретов

## 📁 Структура

```
skills/developer/
├── skill.json          # Конфигурация скила
├── developer.go        # Основной код
├── SECURITY_RULES.md   # Подробные правила
└── README.md           # Этот файл
```

## 🎯 Примеры использования

### Инициализация проекта

```bash
qwen-claw dev init
```

Создаст:
- `.env` с шаблоном конфигурации
- `.gitignore` с правильными правилами

### Проверка перед коммитом

```bash
qwen-claw dev check
```

Вывод:
```
✅ All security checks passed!
```

Или:
```
⚠️ Security issues found:

❌ .env found in git repository!
❌ Possible secret 'token' in logs/app.log
```

## 📖 Документация

Полные правила безопасности см. в [SECURITY_RULES.md](SECURITY_RULES.md)

---

**Версия:** 1.0.0  
**Статус:** ✅ Активен
