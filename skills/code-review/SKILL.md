---
name: code-review
version: 1.0.0
description: Автоматический ревью кода с помощью AI
author: "@sizotov81-hub"
license: MIT
homepage: https://github.com/sizotov81-hub/qwen-claw/tree/main/skills/code-review

type: script
entrypoint: main.sh

commands:
  - code-review
  - review-pr
  - lint

permissions:
  network: false
  filesystem: read-write
  elevated: false

dependencies:
  - git

config:
  max_file_size: 1048576  # 1MB
  extensions:
    - .go
    - .py
    - .js
    - .ts
    - .java
    - .rs

tags:
  - code
  - review
  - lint
  - quality

auto_load: false
min_qwen_claw_version: 1.0.0

changelog:
  - version: 1.0.0
    date: 2026-03-28
    changes:
      - Initial release
      - Анализ Go/Python/JS/TS
      - Проверка стиля кода
---

# Code Review Skill

Автоматический ревью кода с проверкой стиля, потенциальных ошибок и best practices.

## Возможности

- ✅ Статический анализ кода
- ✅ Проверка стиля (gofmt, pylint, eslint)
- ✅ Поиск потенциальных багов
- ✅ Рекомендации по улучшению

## Использование

```bash
# Ревью конкретного файла
qwen-claw skills run code-review path/to/file.go

# Ревью директории
qwen-claw skills run code-review ./src/

# Ревью изменений в Git
qwen-claw skills run code-review --git-diff
```

## Конфигурация

| Параметр | Описание | По умолчанию |
|----------|----------|--------------|
| `max_file_size` | Макс размер файла (байт) | 1MB |
| `extensions` | Поддерживаемые расширения | .go,.py,.js,.ts |

## Примеры

```bash
code-review main.go
review-pr feature-branch
lint ./pkg/
```
