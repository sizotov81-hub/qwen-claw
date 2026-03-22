# Автозагрузка скилов (Auto-Load Skills)

**Версия:** 1.0
**Дата:** 22 марта 2026

---

## 📋 Описание

Функция автозагрузки скилов позволяет автоматически загружать и активировать навыки при старте Qwen-Claw.

---

## ⚙️ Как работает

1. **При старте** `qwen-claw` загружает все скилы из `skills/`
2. **Фильтрация** — выбираются скилы с `auto_load: true`
3. **Уведомление** — показывается список загруженных скилов
4. **Активация** — скилы готовы к использованию

---

## 🔧 Настройка

### Включение автозагрузки для скила

Откройте `skill.json` и добавьте поле `auto_load`:

```json
{
  "name": "developer",
  "description": "Навык разработчика",
  "enabled": true,
  "auto_load": true,
  "type": "system",
  "commands": ["dev", "code"],
  "entry_point": "developer.go"
}
```

### Поля skill.json

| Поле | Тип | Описание |
|------|-----|----------|
| `name` | string | Имя навыка |
| `description` | string | Описание |
| `enabled` | bool | Включён ли навык |
| `auto_load` | bool | **Загружать автоматически при старте** |
| `type` | string | Тип: `builtin`, `external`, `script` |
| `commands` | array | Команды для активации |
| `entry_point` | string | Точка входа (файл) |

---

## 📁 Примеры

### Пример 1: Developer skill (автозагрузка)

```json
{
  "name": "developer",
  "description": "Навык разработчика для работы с кодом",
  "auto_load": true,
  "commands": ["dev", "code", "refactor"]
}
```

### Пример 2: Telegram bot (без автозагрузки)

```json
{
  "name": "telegram-bot",
  "description": "Telegram бот для уведомлений",
  "auto_load": false,
  "commands": ["telegram", "notify"]
}
```

---

## 🚀 Использование

### Старт с автозагрузкой

```bash
cd /home/ss/qwen-claw
./qwen-claw run "привет"
```

**Вывод:**
```
🔹 Auto-loading skills (1):
   ✅ developer (Навык разработчика для работы с кодом проекта qwen-claw)
```

### Просмотр всех скилов

```bash
./qwen-claw skills
```

**Вывод:**
```
📦 Skills (3 total):

✅ [system] developer
   Description: Навык разработчика для работы с кодом проекта qwen-claw
   Commands: dev, develop, code, refactor

✅ [script] example
   Description: Пример навыка для демонстрации
   Commands: example, demo, test

✅ [builtin] shell
   Description: Выполнение shell команд
   Commands: exec, run, shell
```

---

## 📝 Рекомендации

### Какие скилы включать в автозагрузку

**✅ Включать:**
- Базовые навыки (developer, shell, file)
- Часто используемые навыки
- Навыки безопасности

**❌ Не включать:**
- Разовые навыки
- Тестовые навыки
- Навыки с высоким потреблением ресурсов

### Ограничения

- Не более **5 скилов** в автозагрузке
- Общий размер автозагружаемых скилов ≤ **10 MB**
- Время загрузки ≤ **5 секунд**

---

## 🛠️ API

### Методы Engine

```go
// GetAutoLoadSkills возвращает список навыков с включённой автозагрузкой
func (e *Engine) GetAutoLoadSkills() []*Skill {
    result := make([]*Skill, 0)
    for _, skill := range e.skills {
        if skill.AutoLoad && skill.Enabled {
            result = append(result, skill)
        }
    }
    return result
}
```

### Структура Skill

```go
type Skill struct {
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Type        SkillType `json:"type"`
    EntryPoint  string    `json:"entry_point,omitempty"`
    Commands    []string  `json:"commands"`
    Enabled     bool      `json:"enabled"`
    AutoLoad    bool      `json:"auto_load,omitempty"`
}
```

---

## 📊 Логирование

Автозагрузка скилов логируется в stdout:

```
🔹 Auto-loading skills (N):
   ✅ <name> (<description>)
   ✅ <name> (<description>)
```

---

## 🔍 Отладка

### Включить подробный вывод

```bash
./qwen-claw -v run "тест"
```

### Проверить загрузку скилов

```bash
./qwen-claw skills
```

### Проверить конкретный скил

```bash
./qwen-claw skills info developer
```

---

## ⚠️ Возможные проблемы

### Проблема 1: Скил не загружается

**Причина:** `auto_load: false` или не указан

**Решение:**
```json
{
  "auto_load": true
}
```

### Проблема 2: Ошибка при загрузке

**Причина:** Неверный путь к `entry_point`

**Решение:** Проверить наличие файла:
```bash
ls -la skills/developer/developer.go
```

### Проблема 3: Скил загружается, но не работает

**Причина:** `enabled: false`

**Решение:**
```json
{
  "enabled": true
}
```

---

## 📚 Связанные документы

- [skill-management.md](../../.qwen/skill-management.md) — Управление скилами
- [skills/developer/skill.json](../skills/developer/skill.json) — Пример навыка
- [internal/skills/engine.go](../internal/skills/engine.go) — Движок скилов

---

**END OF DOCUMENT**
