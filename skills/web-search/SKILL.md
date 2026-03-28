---
name: web-search
version: 1.0.0
description: Поиск информации в интернете через DuckDuckGo и другие поисковые системы
author: "@sizotov81-hub"
license: MIT
homepage: https://github.com/sizotov81-hub/qwen-claw/tree/main/skills/web-search

# Тип навыка: script, external, builtin
type: script

# Точка входа (файл скрипта или бинарника)
entrypoint: main.sh

# Команды для активации навыка
commands:
  - search-web
  - google
  - duckduckgo
  - web-search

# Разрешения безопасности
permissions:
  # Доступ к сети (true/false)
  network: true
  # Доступ к файловой системе (none/readonly/read-write)
  filesystem: readonly
  # Повышенные привилегии (требуется для опасных операций)
  elevated: false
  # Разрешённые команды (для network: список хостов, для elevated: список команд)
  allowlist:
    - "api.duckduckgo.com"
    - "html.duckduckgo.com"

# Зависимости (внешние утилиты)
dependencies:
  - curl
  - jq

# Конфигурация по умолчанию
config:
  # Таймаут запроса в секундах
  timeout: 30
  # Количество попыток при ошибке
  retry: 3
  # Поисковый движок по умолчанию
  default_engine: duckduckgo
  # Максимум результатов
  max_results: 10
  # Кэширование результатов (сек)
  cache_ttl: 3600

# Теги для поиска в реестре
tags:
  - search
  - web
  - information
  - research

# Автозагрузка при старте (true/false)
auto_load: false

# Минимальная версия Qwen-Claw
min_qwen_claw_version: 1.0.0

# changelog изменений
changelog:
  - version: 1.0.0
    date: 2026-03-28
    changes:
      - Initial release
      - DuckDuckGo интеграция
      - Поддержка кэширования
---

# Web Search Skill

Поиск информации в интернете через DuckDuckGo HTML API.

## Возможности

- ✅ Поиск через DuckDuckGo
- ✅ Поддержка нескольких поисковых движков
- ✅ Кэширование результатов
- ✅ Форматированный вывод

## Использование

### Через CLI:

```bash
qwen-claw skills run web-search "Golang microservices"
```

### Через команды:

```bash
search-web "best AI frameworks 2026"
google "Qwen Code CLI documentation"
duckduckgo "open source AI agents"
```

## Конфигурация

| Параметр | Описание | По умолчанию |
|----------|----------|--------------|
| `timeout` | Таймаут запроса (сек) | 30 |
| `retry` | Количество попыток | 3 |
| `default_engine` | Поисковый движок | duckduckgo |
| `max_results` | Максимум результатов | 10 |
| `cache_ttl` | Время кэширования (сек) | 3600 |

## Примеры

### Простой поиск:

```
search-web "Python async best practices"
```

### Поиск с ограничением по времени:

```
search-web "Golang 1.25 features" --engine=google --max=5
```

## Разрешения

- **network**: требуется для HTTP запросов к поисковым API
- **filesystem**: readonly для кэширования результатов

## Безопасность

Навык выполняется в песочнице с ограниченным доступом:
- Только HTTPS запросы
- Запрещены произвольные команды
- Результаты кэшируются во временной директории

## Зависимости

- `curl` - для HTTP запросов
- `jq` - для парсинга JSON ответов

## Лицензия

MIT License
