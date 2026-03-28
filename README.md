# 🤖 Qwen-Claw Microservices

**AI-помощник с микросервисной архитектурой**

[![Status](https://img.shields.io/badge/status-production%20ready-green)](.)
[![Version](https://img.shields.io/badge/version-1.0.0-blue)](.)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sizotov81-hub/qwen-claw)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

---

## 🚀 Быстрый старт

### 1. Запустить все сервисы:

```bash
./start-all.sh
```

### 2. Открыть веб-интерфейс:

```
http://localhost:64656
```

### 3. Проверить статус:

```bash
./qwen-claw-cli status
```

---

## 🏗️ Архитектура

### Микросервисы:

| Сервис | Порт | Описание |
|--------|------|----------|
| **API Gateway** | 55050 (gRPC), 58080 (HTTP) | Оркестрация, REST/gRPC |
| **Chat API** | 58085 | Обработка chat запросов |
| **Session Memory** | 55051 | Управление сессиями и памятью |
| **Qwen Wrapper** | 55052 | Обёртка для Qwen Code CLI |
| **LLM Proxy** | 55053 | Прокси для облачных LLM |
| **Tools Executor** | 55054 | Выполнение инструментов |
| **Web UI** | 64656 | Веб-интерфейс |

### Схема:

```
┌─────────────────────────────────────────────────────────┐
│                    Клиенты                               │
│  Web UI (64656) │ CLI │ Telegram │ Mobile               │
└─────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────┐
│                   API Gateway                            │
│  gRPC (55050) │ HTTP (58080) │ Chat API (58085)         │
└─────────────────────────────────────────────────────────┘
        │              │              │
        ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│   Session    │ │    Qwen      │ │     LLM      │
│   Memory     │ │   Wrapper    │ │    Proxy     │
│   (55051)    │ │   (55052)    │ │   (55053)    │
└──────────────┘ └──────────────┘ └──────────────┘
                              │
                              ▼
                    ┌──────────────┐
                    │    Tools     │
                    │   Executor   │
                    │   (55054)    │
                    └──────────────┘
```

---

## 📦 Установка

### Требования:

- Go 1.25+
- Docker (опционально)
- Qwen Code CLI (опционально)

### Из исходников:

```bash
git clone https://github.com/sizotov81-hub/qwen-claw.git
cd qwen-claw
go build -o qwen-claw ./cmd/main.go
```

### Сборка всех сервисов:

```bash
CGO_ENABLED=0 go build -o bin/api-gateway ./cmd/api-gateway
CGO_ENABLED=0 go build -o bin/session-memory ./cmd/session-memory
CGO_ENABLED=0 go build -o bin/qwen-wrapper ./cmd/qwen-wrapper
CGO_ENABLED=0 go build -o bin/llm-proxy ./cmd/llm-proxy
CGO_ENABLED=0 go build -o bin/tools-executor ./cmd/tools-executor
CGO_ENABLED=0 go build -o bin/webui ./cmd/webui
```

---

## 🎮 Использование

### CLI:

```bash
# Запустить все сервисы
./start-all.sh

# Проверить статус
./qwen-claw-cli status

# Health check
./qwen-claw-cli health

# Логи
./qwen-claw-cli logs api-gateway

# Остановить сервисы
./stop-all.sh
```

### Веб-интерфейс:

```
http://localhost:64656
```

### Тестовые страницы:

- **Основная:** http://localhost:64656/
- **Минимум:** http://localhost:64656/minimum.html
- **Тест 2:** http://localhost:64656/test2.html

---

## 🧪 Тестирование

```bash
# Все тесты
./test-all.sh

# Unit тесты
go test ./...

# Integration тесты
go test ./tests/integration/...

# Load тесты (k6)
k6 run tests/load/load_test.js

# Покрытие
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Покрытие:

| Пакет | Покрытие |
|-------|----------|
| internal/session | 100% |
| internal/client | 100% |
| internal/rabbitmq | 100% |
| internal/circuitbreaker | 100% |
| internal/server | 100% |
| pkg/config | 94.3% |
| **Всего** | **94.3%** |

---

## 📊 Мониторинг

### Health endpoints:

```bash
curl http://localhost:58080/health          # API Gateway
curl http://localhost:58081/health          # Session Memory
curl http://localhost:58082/health          # Qwen Wrapper
curl http://localhost:58083/health          # LLM Proxy
curl http://localhost:58084/health          # Tools Executor
curl http://localhost:64656/api/health     # Web UI
```

### Metrics:

```bash
curl http://localhost:59090/metrics        # API Gateway
curl http://localhost:59091/metrics        # Session Memory
curl http://localhost:59092/metrics        # Qwen Wrapper
curl http://localhost:59093/metrics        # LLM Proxy
curl http://localhost:59094/metrics        # Tools Executor
```

### Chat API:

```bash
curl -X POST http://localhost:58085/api/chat \
  -H "Content-Type: application/json" \
  -d '{"content":"привет"}'
```

---

## 🔧 Конфигурация

### Переменные окружения:

```bash
# Сервисы
SERVICE_NAME=api-gateway
LOG_LEVEL=info
LOG_FORMAT=json

# Database
DATABASE_URL=postgres://user:pass@host:5432/db

# Redis
REDIS_URL=redis://localhost:6379

# RabbitMQ
RABBITMQ_URL=amqp://user:pass@localhost:5672/

# Tracing
JAEGER_ENDPOINT=jaeger:4317
```

### Конфигурационный файл:

```yaml
server:
  grpc_port: 55051
  http_port: 58080
  host: "0.0.0.0"

logger:
  level: info
  format: json

database:
  url: "postgres://..."
  max_open_conns: 25

redis:
  url: "redis://..."
```

---

## 📁 Структура проекта:

```
qwen-claw/
├── cmd/                       # Точки входа сервисов
│   ├── api-gateway/
│   ├── session-memory/
│   ├── qwen-wrapper/
│   ├── llm-proxy/
│   ├── tools-executor/
│   ├── webui/
│   └── main.go                # Оригинальный CLI
├── internal/                  # Внутренняя логика
│   ├── cache/
│   ├── circuitbreaker/
│   ├── client/
│   ├── config/
│   ├── gateway/
│   ├── logger/
│   ├── memory/
│   ├── metrics/
│   ├── rabbitmq/
│   ├── scheduler/
│   ├── server/
│   ├── session/
│   ├── skills/
│   └── web/
├── pkg/                       # Public API
│   ├── api/
│   └── config/
├── deployments/               # Деплой
│   ├── docker/
│   ├── helm/
│   └── k8s/
├── tests/                     # Тесты
│   ├── integration/
│   └── load/
├── web-new/                   # Веб-интерфейс
│   ├── index.html
│   ├── minimum.html
│   ├── test2.html
│   └── app.js
├── scripts/                   # Скрипты
│   ├── start-all.sh
│   ├── stop-all.sh
│   └── test-all.sh
└── docs/                      # Документация
```

---

## 🛡️ Безопасность

### ✅ Делайте:

1. Используйте `.env` для секретов
2. Не коммитьте `.env` в git
3. Используйте HTTPS в продакшене
4. Включите аутентификацию

### ❌ Не делайте:

1. Не храните секреты в коде
2. Не логируйте токены
3. Не используйте HTTP в продакшене

---

## 📚 Документация:

| Документ | Описание |
|----------|----------|
| [MICROSERVICES_ARCHITECTURE.md](MICROSERVICES_ARCHITECTURE.md) | Архитектура микросервисов |
| [CHANGELOG.md](CHANGELOG.md) | История изменений |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Вклад в проект |
| [SECURITY.md](SECURITY.md) | Безопасность |
| [PROJECT_FINAL_REPORT.md](PROJECT_FINAL_REPORT.md) | Финальный отчёт |
| [tests/load/README.md](tests/load/README.md) | Load тесты |

---

## 🤝 Вклад в проект

1. Fork репозиторий
2. Создайте feature branch
3. Commit изменения
4. Push в branch
5. Откройте Pull Request

См. [CONTRIBUTING.md](CONTRIBUTING.md)

---

## 📄 Лицензия

MIT License — см. [LICENSE](LICENSE)

---

## 👥 Авторы

- [@sizotov81-hub](https://github.com/sizotov81-hub)

---

**Qwen-Claw** — микросервисный AI-помощник 🚀
