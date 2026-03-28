---
name: github-tools
version: 1.0.0
description: Инструменты для работы с GitHub API
author: "@sizotov81-hub"
license: MIT
homepage: https://github.com/sizotov81-hub/qwen-claw/tree/main/skills/github-tools

type: script
entrypoint: main.sh

commands:
  - gh
  - github
  - create-pr
  - list-issues

permissions:
  network: true
  filesystem: readonly
  elevated: false

dependencies:
  - curl
  - jq

config:
  timeout: 30
  api_url: https://api.github.com

tags:
  - github
  - git
  - pull-request
  - issues

auto_load: false
min_qwen_claw_version: 1.0.0

changelog:
  - version: 1.0.0
    date: 2026-03-28
    changes:
      - Initial release
      - Создание PR
      - Список issues
      - Информация о репозитории
---

# GitHub Tools Skill

Инструменты для работы с GitHub API.

## Возможности

- ✅ Создание Pull Requests
- ✅ Список Issues
- ✅ Информация о репозитории
- ✅ Управление labels

## Использование

```bash
# Информация о репозитории
qwen-claw skills run github-tools repo sizotov81-hub/qwen-claw

# Список issues
qwen-claw skills run github-tools issues sizotov81-hub/qwen-claw

# Создание PR
qwen-claw skills run github-tools create-pr sizotov81-hub/qwen-claw feature-branch main "Fix bug"
```

## Конфигурация

Требуется GitHub token в переменной окружения:
```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
```
