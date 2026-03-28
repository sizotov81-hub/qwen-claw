# 🛠️ Система навыков Qwen-Claw

**Версия:** 2.0.0
**Дата:** 2026-03-28

---

## 📋 Обзор

Система навыков позволяет расширять возможности Qwen-Claw через модульные плагины.

### Типы навыков:

| Тип | Описание | Примеры |
|-----|----------|---------|
| **builtin** | Встроенные на Go | shell, file, search, memory, git, http |
| **script** | Скрипты (bash/python) | web-search, code-review |
| **external** | Компилируемые бинарники | custom-tools |

---

## 📁 Структура навыка

```
skills/
├── {skill-name}/
│   ├── SKILL.md           # Манифест навыка (обязательно)
│   ├── main.sh            # Точка входа (для script)
│   ├── main.py            # Или Python скрипт
│   ├── README.md          # Документация
│   ├── config.yaml        # Конфигурация (опционально)
│   └── tests/             # Тесты (опционально)
```

---

## 📄 Формат SKILL.md

```markdown
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

dependencies:
  - curl
  - jq

config:
  timeout: 30
  retry: 3
  default_engine: duckduckgo

tags:
  - search
  - web
  - information
---

# Web Search Skill

Поиск информации в интернете через DuckDuckGo API.

## Использование

```bash
qwen-claw skills run web-search "query"
```

## Примеры

```
search-web "Golang microservices"
google "best AI frameworks 2026"
```

## Конфигурация

| Параметр | Описание | По умолчанию |
|----------|----------|--------------|
| `timeout` | Таймаут запроса (сек) | 30 |
| `retry` | Количество попыток | 3 |
| `default_engine` | Поисковый движок | duckduckgo |

## Разрешения

- **network**: требуется для HTTP запросов
- **filesystem**: только чтение для кэша

## Безопасность

Навык выполняется в песочнице с ограниченным доступом к сети.
```

---

## 🔧 Управление навыками

### CLI команды:

```bash
# Список навыков
qwen-claw skills list

# Информация о навыке
qwen-claw skills info web-search

# Запустить навык
qwen-claw skills run web-search "query"

# Установить из реестра
qwen-claw skills install skill-name

# Удалить навык
qwen-claw skills uninstall skill-name

# Включить/отключить
qwen-claw skills enable web-search
qwen-claw skills disable web-search

# Поиск в реестре
qwen-claw skills search "search"

# Популярные навыки
qwen-claw skills popular

# Проверить обновления
qwen-claw skills update
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
qwen-claw skills info my-skill
qwen-claw skills run my-skill arg1 arg2
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

| Разрешение | Описание |
|------------|----------|
| **network** | Доступ к сети |
| **filesystem** | Доступ к файлам (readonly/read-write/none) |
| **elevated** | Повышенные привилегии |

### Аудит:

Все вызовы навыков логируются:

```bash
qwen-claw skills audit
```

---

## 📚 Реестр навыков

### Локальный реестр:

```bash
# Инициализировать
qwen-claw skills registry init

# Добавить навык
qwen-claw skills registry add skills/my-skill

# Опубликовать
qwen-claw skills registry publish
```

### Установка из реестра:

```bash
qwen-claw skills install web-search
```

---

**Система навыков Qwen-Claw** — расширяйте возможности вашего AI-помощника! 🚀
