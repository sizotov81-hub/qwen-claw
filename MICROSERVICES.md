# 🏗️ Микросервисная архитектура Qwen-Claw

**Версия:** 2.0.0
**Дата:** 2026-03-28
**Статус:** ✅ Независимые микросервисы

---

## 📋 Обзор

Qwen-Claw теперь состоит из **независимых микросервисов**, каждый из которых:
- ✅ Имеет собственный `go.mod`
- ✅ Собирается отдельно
- ✅ Развёртывается независимо
- ✅ Работает автономно (если один упал - остальные работают)
- ✅ Общается с другими **только** через gRPC/HTTP

---

## 🎯 Микросервисы

| Сервис | Порт gRPC | Порт HTTP | Порт Metrics | Описание |
|--------|-----------|-----------|--------------|----------|
| **api-gateway** | 55050 | 58080 | 59090 | Маршрутизация, gRPC ↔ HTTP |
| **session-memory** | 55051 | 58081 | 59091 | Сессии, память, кэш |
| **qwen-wrapper** | 55052 | 58082 | 59092 | Интеграция с Qwen Code CLI |
| **llm-proxy** | 55053 | 58083 | 59093 | Прокси для облачных LLM |
| **tools-executor** | 55054 | 58084 | 59094 | Выполнение инструментов |

---

## 📁 Структура проекта

```
qwen-claw/
├── cmd/                          # Точки входа сервисов
│   ├── api-gateway/
│   │   ├── main.go               # Точка входа
│   │   ├── chat.go               # Chat логика
│   │   ├── chat_server.go        # Chat сервер
│   │   └── go.mod                # Независимые зависимости
│   ├── session-memory/
│   │   ├── main.go
│   │   └── go.mod
│   ├── qwen-wrapper/
│   │   ├── main.go
│   │   └── go.mod
│   ├── llm-proxy/
│   │   ├── main.go
│   │   └── go.mod
│   ├── tools-executor/
│   │   ├── main.go
│   │   └── go.mod
│   └── webui/
│       └── main.go
├── services/
│   └── common/                   # Общие утилиты
│       ├── logging/              # Логгер (Zap)
│       ├── metrics/              # Prometheus метрики
│       ├── grpcutil/             # Базовый gRPC сервер
│       └── go.mod                # Общие зависимости
├── internal/                     # Бизнес-логика
│   ├── cache/
│   ├── circuitbreaker/
│   ├── client/
│   ├── metrics/
│   ├── rabbitmq/
│   ├── server/
│   └── session/
├── pkg/                          # Public API
│   ├── api/
│   │   └── proto/                # Protocol Buffer определения
│   └── config/                   # Конфигурация
├── bin/                          # Собранные бинарники
│   ├── api-gateway
│   ├── session-memory
│   ├── qwen-wrapper
│   ├── llm-proxy
│   ├── tools-executor
│   └── webui
├── deployments/                  # Деплой
│   ├── docker/
│   │   └── services/             # Dockerfile сервисов
│   └── k8s/                      # Kubernetes манифесты
├── tests/                        # Тесты
│   ├── integration/
│   └── load/
├── web-new/                      # Веб-интерфейс
├── scripts/                      # Скрипты
│   └── generate-proto.sh         # Генерация proto файлов
└── docker-compose*.yml           # Docker Compose конфигурации
```

---

## 🚀 Быстрый старт

### Запуск всех сервисов:

```bash
./start-all.sh
```

### Запуск отдельных сервисов:

```bash
# API Gateway
./start-api-gateway.sh

# Session Memory
./start-session-memory.sh

# Qwen Wrapper
./start-qwen-wrapper.sh

# LLM Proxy
./start-llm-proxy.sh

# Tools Executor
./start-tools-executor.sh
```

### Остановка всех сервисов:

```bash
./stop-all.sh
```

### Переменные окружения:

```bash
# API Gateway
GRPC_PORT=55050 HTTP_PORT=58080 ./start-api-gateway.sh

# Session Memory
DATABASE_URL="postgres://..." REDIS_URL="redis://..." ./start-session-memory.sh

# LLM Proxy
OPENAI_KEY="sk-..." ANTHROPIC_KEY="..." ./start-llm-proxy.sh
```

---

## 🔌 Межсервисное взаимодействие

### gRPC (основное):

```
api-gateway (55050)
    │
    ├──→ session-memory (55051)
    ├──→ qwen-wrapper (55052)
    ├──→ llm-proxy (55053)
    └──→ tools-executor (55054)
```

### HTTP (health/metrics):

```bash
# Health check
curl http://localhost:58080/health          # api-gateway
curl http://localhost:58081/health          # session-memory
curl http://localhost:58082/health          # qwen-wrapper
curl http://localhost:58083/health          # llm-proxy
curl http://localhost:58084/health          # tools-executor

# Metrics
curl http://localhost:59090/metrics         # api-gateway
curl http://localhost:59091/metrics         # session-memory
curl http://localhost:59092/metrics         # qwen-wrapper
curl http://localhost:59093/metrics         # llm-proxy
curl http://localhost:59094/metrics         # tools-executor
```

---

## 📦 Общие утилиты (services/common)

### logging
```go
import "github.com/user/qwen-claw-common/logging"

logging.Init(&logging.Config{
    Level:       "info",
    Format:      "json",
    ServiceName: "api-gateway",
})

logging.Info("Service started")
```

### metrics
```go
import "github.com/user/qwen-claw-common/metrics"

metrics.NewMetrics(&metrics.Config{
    ServiceName: "api-gateway",
    Port:        59090,
    Enabled:     true,
})
```

### grpcutil
```go
import "github.com/user/qwen-claw-common/grpcutil"

server := grpcutil.NewGRPCServer(
    55050, 58080, "0.0.0.0",
    grpcutil.WithLogger(logging.GetLogger()),
)
```

---

## 🛡️ Отказоустойчивость

### Принцип:
- ❌ **Нет общих зависимостей** между сервисами (кроме common)
- ❌ **Нет прямого импорта** одного сервиса в другом
- ✅ **Общение только через сеть** (gRPC/HTTP)
- ✅ **Каждый сервис работает в своём процессе**

### Если сервис упал:

```
session-memory упал → api-gateway продолжает работать
                      (возвращает ошибки при запросе к session-memory)
```

### Graceful Shutdown:

Каждый сервис корректно завершает работу:
1. Перестаёт принимать новые запросы
2. Завершает текущие запросы (30 сек timeout)
3. Закрывает подключения
4. Логирует остановку

---

## 🧪 Сборка

### Сборка всех сервисов:

```bash
# Common модуль
cd services/common && go mod tidy

# API Gateway
cd cmd/api-gateway && CGO_ENABLED=0 go build -o ../../bin/api-gateway .

# Session Memory
cd cmd/session-memory && CGO_ENABLED=0 go build -o ../../bin/session-memory .

# И так далее...
```

### Кросс-компиляция:

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/api-gateway .

# macOS ARM64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bin/api-gateway .

# Windows AMD64
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/api-gateway.exe .
```

---

## 📊 Мониторинг

### Health Check:

```bash
# Проверка всех сервисов
for port in 58080 58081 58082 58083 58084; do
    curl -s http://localhost:$port/health | jq
done
```

### Metrics (Prometheus):

```bash
# Request rate
rate(http_requests_total{service="api-gateway"}[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# p95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

---

## 🔧 Конфигурация

### Переменные окружения:

| Переменная | Сервис | Описание |
|------------|--------|----------|
| `GRPC_PORT` | Все | gRPC порт |
| `HTTP_PORT` | Все | HTTP порт |
| `METRICS_PORT` | Все | Порт метрик |
| `HOST` | Все | Host для прослушивания |
| `LOG_LEVEL` | Все | Уровень логирования |
| `LOG_FORMAT` | Все | Формат логов (json/console) |
| `DATABASE_URL` | session-memory | PostgreSQL URL |
| `REDIS_URL` | session-memory | Redis URL |
| `OPENAI_KEY` | llm-proxy | OpenAI API key |
| `ANTHROPIC_KEY` | llm-proxy | Anthropic API key |
| `DASHSCOPE_KEY` | llm-proxy | DashScope API key |
| `QWEN_PATH` | qwen-wrapper | Путь к Qwen CLI |
| `SANDBOX_MODE` | tools-executor | Режим песочницы |

---

## 📈 Масштабирование

### Горизонтальное:

```bash
# Запуск нескольких реплик session-memory
./start-session-memory.sh &  # Порт 55051
GRPC_PORT=55055 ./start-session-memory.sh &  # Порт 55055

# Load balancer распределяет запросы
```

### Kubernetes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: session-memory
spec:
  replicas: 3
  selector:
    matchLabels:
      app: session-memory
  template:
    spec:
      containers:
      - name: session-memory
        image: qwen-claw/session-memory:latest
        ports:
        - containerPort: 55051
```

---

## ✅ Чеклист независимости

- [x] Каждый сервис имеет свой `go.mod`
- [x] Нет импортов между сервисами
- [x] Общие утилиты выделены в `services/common`
- [x] Общение только через gRPC/HTTP
- [x] Каждый сервис собирается отдельно
- [x] Graceful shutdown в каждом сервисе
- [x] Health check в каждом сервисе
- [x] Metrics в каждом сервисе

---

**Qwen-Claw Microservices** — AI-помощник с независимыми микросервисами 🚀
