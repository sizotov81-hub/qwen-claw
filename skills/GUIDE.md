# 🛠️ Система навыков Qwen-Claw — Руководство

**Версия:** 2.0.0
**Дата:** 2026-03-28
**Статус:** ✅ Готово к использованию

---

## 📋 Обзор

Система навыков позволяет расширять возможности Qwen-Claw через модульные плагины с поддержкой:
- ✅ Формат манифеста SKILL.md
- ✅ Песочница (Docker)
- ✅ Система разрешений
- ✅ Аудит выполнения
- ✅ CLI управление

---

## 🚀 Быстрый старт

### Список навыков:

```bash
qwen-claw skills list
```

### Запуск навыка:

```bash
qwen-claw skills run web-search "Golang microservices"
```

### Информация о навыке:

```bash
qwen-claw skills info web-search
```

---

## 📁 Структура навыка

```
skills/
├── {skill-name}/
│   ├── SKILL.md           # Манифест (обязательно)
│   ├── main.sh            # Скрипт (для type: script)
│   ├── README.md          # Документация
│   └── tests/             # Тесты (опционально)
```

---

## 📄 Формат SKILL.md

```yaml
---
name: web-search
version: 1.0.0
description: Поиск информации в интернете
author: "@sizotov81-hub"
license: MIT

type: script
entrypoint: main.sh

commands:
  - search-web
  - google
  - duckduckgo

permissions:
  network: true
  filesystem: readonly
  elevated: false
  allowlist:
    - "api.duckduckgo.com"

dependencies:
  - curl
  - jq

config:
  timeout: 30
  retry: 3

tags:
  - search
  - web
  - information

auto_load: false
min_qwen_claw_version: 1.0.0
---

# Описание навыка

Подробная документация...
```

---

## 🔧 CLI команды

### list — Список навыков

```bash
# Все навыки
qwen-claw skills list

# С флагами
qwen-claw skills list --verbose
```

### info — Информация о навыке

```bash
qwen-claw skills info web-search
```

### run — Выполнить навык

```bash
# Базовый запуск
qwen-claw skills run web-search "query"

# С опциями
qwen-claw skills run web-search "query" --isolation=all --force

# С аудитом
qwen-claw skills run web-search "query" --audit-log=/var/log/skills.log
```

### install — Установить навык

```bash
qwen-claw skills install skill-name
```

### uninstall — Удалить навык

```bash
qwen-claw skills uninstall skill-name
```

### validate — Валидировать навык

```bash
qwen-claw skills validate web-search
```

### audit — Журнал аудита

```bash
# Последние записи
qwen-claw skills audit

# С фильтром
qwen-claw skills audit --skill=web-search --limit=10
```

### registry — Управление реестром

```bash
# Инициализировать
qwen-claw skills registry init
```

---

## 🔒 Безопасность

### Уровни изоляции:

| Режим | Описание |
|-------|----------|
| **off** | Без изоляции |
| **non-main** (default) | Групповые чаты в контейнере |
| **all** | Все вызовы в контейнере |

### Разрешения:

| Разрешение | Описание | Значения |
|------------|----------|----------|
| **network** | Доступ к сети | true/false |
| **filesystem** | Доступ к файлам | none/readonly/read-write |
| **elevated** | Повышенные привилегии | true/false |

### Аудит:

Все вызовы логируются в JSONL формат:

```json
{
  "timestamp": "2026-03-28T12:00:00Z",
  "skill_name": "web-search",
  "command": "/path/to/main.sh",
  "args": ["query"],
  "success": true,
  "duration": 1500000000,
  "isolation_level": "non-main",
  "permissions": {
    "network": true,
    "filesystem": "readonly",
    "elevated": false
  }
}
```

---

## 📦 Встроенные навыки

| Навык | Команды | Описание |
|-------|---------|----------|
| **shell** | exec, run, shell | Выполнение shell команд |
| **file** | read, write, edit | Операции с файлами |
| **search** | search, grep, find | Поиск по коду |
| **memory** | remember, recall | Управление памятью |
| **git** | git, commit, push | Git операции |
| **http** | http, curl, get, post | HTTP запросы |
| **notify** | notify, alert | Уведомления |

---

## 🎁 Примеры навыков

### web-search

Поиск в интернете через DuckDuckGo:

```bash
qwen-claw skills run web-search "Golang best practices"
```

### code-review

Автоматический ревью кода:

```bash
qwen-claw skills run code-review main.go
qwen-claw skills run code-review --git-diff
```

### github-tools

Работа с GitHub API:

```bash
export GITHUB_TOKEN=ghp_xxx

# Информация о репозитории
qwen-claw skills run github-tools repo sizotov81-hub/qwen-claw

# Список issues
qwen-claw skills run github-tools issues sizotov81-hub/qwen-claw

# Создание PR
qwen-claw skills run github-tools create-pr repo feature main "Fix bug"
```

---

## 🚀 Создание своего навыка

### 1. Создайте директорию:

```bash
mkdir -p skills/my-skill
```

### 2. Создайте SKILL.md:

```markdown
---
name: my-skill
version: 1.0.0
description: Мой первый навык
author: "@username"
type: script
entrypoint: main.sh
commands:
  - my-command
permissions:
  network: false
  filesystem: readonly
  elevated: false
---

# My Skill

Описание навыка.
```

### 3. Создайте скрипт:

```bash
#!/bin/bash
# main.sh

echo "My skill executed with args: $@"
```

### 4. Сделайте исполняемым:

```bash
chmod +x skills/my-skill/main.sh
```

### 5. Проверьте:

```bash
# Валидация
qwen-claw skills validate my-skill

# Запуск
qwen-claw skills run my-skill arg1 arg2

# Информация
qwen-claw skills info my-skill
```

---

## 🧪 Тестирование навыков

### Тестирование скрипта:

```bash
cd skills/my-skill

# Прямой запуск
./main.sh test-arg

# Через CLI
qwen-claw skills run my-skill test-arg
```

### Валидация:

```bash
# Проверка формата
qwen-claw skills validate my-skill

# Проверка с force
qwen-claw skills run my-skill --force
```

---

## 🔗 Ссылки

- [ROADMAP.md](../ROADMAP.md) — Дорожная карта
- [README.md](../README.md) — Основная документация
- [MICROSERVICES_ARCHITECTURE.md](../MICROSERVICES_ARCHITECTURE.md) — Архитектура

---

**Система навыков Qwen-Claw** — расширяйте возможности вашего AI-помощника! 🚀
