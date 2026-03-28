# 🔍 Web Search Skill

Поиск информации в интернете через DuckDuckGo и другие поисковые системы.

## Возможности

- ✅ Поиск через DuckDuckGo HTML API
- ✅ Поддержка Google (с ограничениями)
- ✅ Кэширование результатов
- ✅ Форматированный вывод
- ✅ Несколько поисковых движков

## Установка

```bash
# Навык уже установлен в директории skills/web-search
# Убедитесь, что зависимости установлены:
sudo apt install curl jq  # Ubuntu/Debian
brew install curl jq      # macOS
```

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

### С опциями:

```bash
# Поиск через DuckDuckGo (по умолчанию)
qwen-claw skills run web-search "Python async"

# Поиск через Google
qwen-claw skills run web-search "AI agents" --engine=google

# Ограничить результаты
qwen-claw skills run web-search "microservices" --max=5

# Без кэша
qwen-claw skills run web-search "fresh results" --nocache
```

## Конфигурация

Переменные окружения:

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `TIMEOUT` | Таймаут запроса (сек) | 30 |
| `MAX_RESULTS` | Максимум результатов | 10 |
| `ENGINE` | Поисковый движок | duckduckgo |
| `CACHE_TTL` | Время кэширования (сек) | 3600 |
| `CACHE_DIR` | Директория кэша | `~/.qwen-claw/skills/cache` |

## Примеры вывода

```
════════════════════════════════════════
  Результаты поиска: Golang microservices
════════════════════════════════════════

 1. https://go.dev/doc/tutorial/create-service
 2. https://github.com/golang-standards/project-layout
 3. https://microservices.io/patterns/microservices.html
 4. https://aws.amazon.com/microservices/
 5. https://cloud.google.com/architecture/microservices

════════════════════════════════════════
```

## Разрешения

- **network**: требуется для HTTP запросов
- **filesystem**: readonly для кэширования

## Безопасность

- Выполняется в песочнице
- Только HTTPS запросы
- Запрещены произвольные команды
- Результаты кэшируются во временной директории

## Зависимости

- `curl` - для HTTP запросов
- `jq` - для парсинга JSON (опционально)
- `python3` - для URL encoding

## Лицензия

MIT License
