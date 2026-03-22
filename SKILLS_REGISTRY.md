# 🧩 Skills Registry — Система навыков с реестром

**Вдохновлено ClawHub из OpenClaw**

---

## 🎯 Обзор

Skills Registry — это система управления навыками с централизованным реестром, селективной инъекцией и CLI для установки/удаления.

### Ключевые возможности:

- ✅ **Реестр навыков** — поиск, установка, обновление
- ✅ **Селективная инъекция** — только релевантные навыки для запроса
- ✅ **CLI управление** — install/uninstall/search/enable/disable
- ✅ **Кэширование** — кэш реестра для ускорения поиска
- ✅ **Проверка обновлений** — уведомления о новых версиях

---

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────┐
│                   Skill Engine                      │
├─────────────────────────────────────────────────────┤
│  injector      → Селективная загрузка навыков       │
│  registry      → Клиент реестра (ClawHub-like)      │
│  skills        → Загруженные навыки                 │
└─────────────────────────────────────────────────────┘
```

---

## 📁 Компоненты

### 1. RegistryClient

Клиент для работы с реестром навыков.

**Функции:**
- `Search(query, category, tags)` — поиск навыков
- `GetSkill(name)` — информация о навыке
- `Download(name, destDir)` — скачивание навыка
- `ListPopular(limit)` — популярные навыки
- `ListCategories()` — список категорий

**Конфигурация:**
```go
type RegistryConfig struct {
    BaseURL    string        // URL реестра
    Timeout    time.Duration // Таймаут запросов
    CacheDir   string        // Директория кэша
    CacheTTL   time.Duration // Время жизни кэша
}
```

### 2. SkillInjector

Инжектор для селективной загрузки навыков.

**Функции:**
- `InjectSkills(query)` — найти релевантные навыки
- `GetSkillPrompt(skill)` — получить промпт навыка
- `FormatSkillsForPrompt(skills)` — форматировать для промпта
- `InstallSkill(name, client)` — установить навык
- `EnableSkill(name)` / `DisableSkill(name)` — управление

**Алгоритм релевантности:**
```
Совпадение с командами:    +0.5
Совпадение с описанием:    +0.3
Совпадение с именем:       +0.4
Совпадение ключевых слов:  +0.1 за каждое

Порог: 0.3 (30%)
Максимум навыков: 5
```

### 3. Skill Engine

Расширенный движок навыков.

**Новые методы:**
- `InjectSkills(query)` — селективная загрузка
- `InstallSkill(name)` — установка из реестра
- `UninstallSkill(name)` — удаление
- `SearchSkills(query)` — поиск в реестре
- `ListPopularSkills(limit)` — популярные
- `CheckForUpdates()` — проверка обновлений

---

## 🚀 CLI команды

### Поиск навыков

```bash
# Поиск по запросу
qwen-claw skills search github

# Поиск по категории
qwen-claw skills search --category development

# Поиск по тегам
qwen-claw skills search --tags git,ci
```

### Установка/удаление

```bash
# Установить навык
qwen-claw skills install github

# Удалить навык
qwen-claw skills uninstall github
```

### Управление

```bash
# Включить навык
qwen-claw skills enable github

# Отключить навык
qwen-claw skills disable github

# Информация о навыке
qwen-claw skills info github
```

### Популярные и обновления

```bash
# Популярные навыки
qwen-claw skills popular
qwen-claw skills popular 20  # Топ-20

# Проверка обновлений
qwen-claw skills update
```

### Список навыков

```bash
# Все установленные навыки
qwen-claw skills

# Подробный список
qwen-claw skills --verbose
```

---

## 📊 Реестр навыков

### Формат ответа реестра

```json
{
  "skills": [
    {
      "name": "github",
      "description": "GitHub integration for PR reviews, issues, CI/CD",
      "version": "1.2.0",
      "author": "qwen-claw",
      "category": "development",
      "tags": ["git", "ci", "pr"],
      "repository": "https://github.com/qwen-claw/skill-github",
      "download_url": "https://registry.qwen-claw.dev/skills/github.zip",
      "homepage": "https://qwen-claw.dev/skills/github",
      "license": "MIT",
      "downloads": 15420,
      "created": "2026-01-15T10:00:00Z",
      "updated": "2026-03-20T14:30:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### Категории навыков

| Категория | Описание | Примеры |
|-----------|----------|---------|
| `development` | Разработка | github, docker, kubernetes |
| `communication` | Коммуникация | telegram, slack, discord |
| `automation` | Автоматизация | cron, webhook, workflow |
| `analysis` | Анализ | analyzer, linter, security |
| `utilities` | Утилиты | http, file, shell |

---

## 🧪 Селективная инъекция

### Как это работает

1. Пользователь отправляет запрос
2. Injector вычисляет релевантность каждого навыка
3. Выбираются топ-N навыков (порог 30%)
4. Навыки инжектятся в промпт агента

### Пример

**Запрос:** "Сделай PR review"

**Релевантность:**
| Навык | Score | Инжектится |
|-------|-------|-----------|
| github | 0.9 | ✅ |
| code-review | 0.7 | ✅ |
| git | 0.5 | ✅ |
| telegram | 0.1 | ❌ |

**Промпт агента:**
```
Ты — Джарвис, AI-помощник.

## Available Skills

# github
GitHub integration for PR reviews, issues, CI/CD...

# code-review
Code review best practices...

# git
Git version control...

---

Запрос пользователя: Сделай PR review
```

---

## 📁 Структура навыка

```
skills/<name>/
├── skill.json          # Метаданные
├── SKILL.md            # Промпт/документация
├── PROMPT.md           # Дополнительный промпт (опционально)
├── tools/              # Инструменты (опционально)
├── scripts/            # Скрипты (опционально)
└── tests/              # Тесты (опционально)
```

### skill.json

```json
{
  "name": "github",
  "description": "GitHub integration",
  "version": "1.0.0",
  "enabled": true,
  "auto_load": false,
  "type": "external",
  "commands": ["github", "gh", "pr", "review"],
  "entry_point": "github.sh",
  "author": "qwen-claw",
  "category": "development",
  "tags": ["git", "ci", "pr"]
}
```

---

## 🔧 Настройка

### Изменение URL реестра

```go
// В main.go или конфиге
registryConfig := &skills.RegistryConfig{
    BaseURL: "https://my-registry.example.com",
    Timeout: 60 * time.Second,
}
```

### Настройка инжектора

```go
injectorConfig := &skills.InjectorConfig{
    SkillsDir: "~/.qwen/skills",
    RelevanceThreshold: 0.5,  // Более строгий порог
    MaxSkillsPerQuery: 3,     // Меньше навыков
}
```

### Кэширование

```go
// Кэш хранится в ~/.qwen/registry-cache/
// TTL по умолчанию: 1 час

registryConfig.CacheTTL = 24 * time.Hour  // 24 часа
```

---

## 🧪 Тестирование

### Проверка работы

```bash
# Посмотреть установленные навыки
qwen-claw skills

# Поиск навыков
qwen-claw skills search github

# Популярные
qwen-claw skills popular

# Проверка обновлений
qwen-claw skills update
```

### Логи

```bash
# Включить debug
qwen-claw skills search github --debug
```

---

## 📚 API для разработчиков

### Поиск навыков

```go
engine := skills.NewEngine(skillsDir)
engine.Load()

// Поиск в реестре
skills, err := engine.SearchSkills("github", "", nil)

// Селективная инъекция
relevantSkills, err := engine.InjectSkills("make PR review")

// Форматирование в промпт
prompt, err := engine.FormatSkillsForPrompt(relevantSkills)
```

### Установка навыка

```go
err := engine.InstallSkill("github")
```

### Проверка обновлений

```go
updates, err := engine.CheckForUpdates()
for _, name := range updates {
    fmt.Printf("Update available: %s\n", name)
}
```

---

## 🎯 Планы развития

- [ ] Реальный реестр (registry.qwen-claw.dev)
- [ ] Веб-интерфейс для просмотра навыков
- [ ] Рейтинги и отзывы
- [ ] Автоматические обновления
- [ ] Зависимости между навыками
- [ ] Песочница для тестирования навыков

---

## 📚 Ссылки

- [OpenClaw ClawHub](https://github.com/openclaw/clawhub)
- [OpenClaw Skills System](https://docs.openclaw.ai/skills)

---

**Skills Registry** — расширяемая система навыков 🚀
