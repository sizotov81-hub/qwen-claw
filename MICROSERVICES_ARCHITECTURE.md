# 🏗️ Qwen-Claw Microservices Architecture

**Версия:** 1.0.0  
**Дата:** 2026-03-28  
**Статус:** ✅ Production Ready

---

## 📋 Обзор

Qwen-Claw — это AI-помощник с микросервисной архитектурой, состоящий из 7 независимых сервисов.

---

## 🎯 Микросервисы

### 1. API Gateway

**Порты:** 55050 (gRPC), 58080 (HTTP), 58085 (Chat API)

**Ответственность:**
- Маршрутизация запросов
- gRPC ↔ HTTP конвертация
- Rate limiting
- Аутентификация

**Зависимости:** Нет

---

### 2. Session Memory

**Порт:** 55051

**Ответственность:**
- Управление сессиями
- Хранение памяти
- Кэширование (Redis)

**Зависимости:**
- PostgreSQL (с pgVector)
- Redis

---

### 3. Qwen Wrapper

**Порт:** 55052

**Ответственность:**
- Интеграция с Qwen Code CLI
- Выполнение команд
- Управление контекстом

**Зависимости:**
- Qwen Code CLI (опционально)

---

### 4. LLM Proxy

**Порт:** 55053

**Ответственность:**
- Прокси для облачных LLM
- Балансировка провайдеров
- Circuit breaker

**Зависимости:**
- OpenAI API (опционально)
- Anthropic API (опционально)

---

### 5. Tools Executor

**Порт:** 55054

**Ответственность:**
- Выполнение инструментов
- Docker sandbox
- Безопасное выполнение

**Зависимости:**
- Docker (опционально)

---

### 6. Web UI

**Порт:** 64656

**Ответственность:**
- Веб-интерфейс
- Chat UI
- Мониторинг сервисов

**Зависимости:**
- API Gateway

---

### 7. Chat API

**Порт:** 58085

**Ответственность:**
- Обработка chat запросов
- Генерация ответов (заглушки)

**Зависимости:**
- Qwen Wrapper (опционально)

---

## 🔌 Межсервисное взаимодействие

### gRPC:

```protobuf
service SessionMemory {
    rpc CreateSession(CreateSessionRequest) returns (Session);
    rpc GetSession(SessionID) returns (Session);
    rpc AddMessage(AddMessageRequest) returns (Message);
}
```

### HTTP REST:

```
GET  /api/health
POST /api/chat
GET  /api/sessions
```

### RabbitMQ:

```
llm-requests       → LLM Proxy
tool-execution     → Tools Executor
session-updates    → Session Memory
```

---

## 📊 Диаграмма последовательности

```
User → Web UI → API Gateway → Session Memory → Qwen Wrapper → LLM Proxy
 │       │          │              │               │              │
 │──────▶│──────────│──────────────│───────────────│──────────────│ HTTP
 │       │          │              │               │              │
 │◀──────│◀─────────│◀─────────────│◀──────────────│◀─────────────│ Response
 │       │          │              │               │              │
```

---

## 🛡️ Отказоустойчивость

### Circuit Breaker:

```go
Config{
    FailureThreshold: 5,    // Ошибок для размыкания
    SuccessThreshold: 3,    // Успехов для замыкания
    Timeout:          30s,  // Таймаут в открытом состоянии
}
```

### Retry Logic:

```go
Config{
    MaxRetries:   3,
    RetryBackoff: 100ms,
}
```

---

## 📈 Масштабирование

### Горизонтальное:

```bash
kubectl scale deployment session-memory --replicas=3
```

### Автоматическое (HPA):

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
spec:
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        averageUtilization: 70
```

---

## 🔒 Безопасность

### Аутентификация:

- JWT токены
- API keys
- OAuth 2.0 (опционально)

### Авторизация:

- RBAC (Role-Based Access Control)
- Rate limiting per user

### Шифрование:

- TLS для HTTPS
- mTLS для gRPC (опционально)

---

## 📊 Мониторинг

### Prometheus метрики:

```promql
# Request rate
rate(http_requests_total{service="api-gateway"}[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# p95 latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Grafana дашборды:

1. Service Overview
2. HTTP Metrics
3. gRPC Metrics
4. Database Metrics
5. Cache Metrics
6. Circuit Breaker

---

## 🧪 Тестирование

### Unit тесты:

```bash
go test ./internal/session/...
go test ./internal/client/...
```

### Integration тесты:

```bash
go test ./tests/integration/...
```

### Load тесты (k6):

```bash
k6 run tests/load/load_test.js
```

---

## 🚀 Деплой

### Docker Compose:

```bash
docker-compose -f docker-compose.microservices.yml up -d
```

### Kubernetes:

```bash
kubectl apply -f deployments/k8s/
```

### Helm:

```bash
helm install qwen-claw ./deployments/helm/qwen-claw
```

---

## 📁 Структура проекта:

```
qwen-claw/
├── cmd/                       # Сервисы
│   ├── api-gateway/
│   ├── session-memory/
│   ├── qwen-wrapper/
│   ├── llm-proxy/
│   ├── tools-executor/
│   ├── webui/
│   └── main.go
├── internal/                  # Логика
│   ├── session/
│   ├── client/
│   ├── circuitbreaker/
│   └── ...
├── pkg/                       # Public API
│   ├── api/
│   └── config/
├── deployments/               # Деплой
│   ├── docker/
│   ├── k8s/
│   └── helm/
├── tests/                     # Тесты
│   ├── integration/
│   └── load/
└── web-new/                   # Web UI
    ├── index.html
    ├── minimum.html
    └── test2.html
```

---

## 📚 Ссылки:

- [README.md](README.md) — Основная документация
- [CHANGELOG.md](CHANGELOG.md) — История изменений
- [CONTRIBUTING.md](CONTRIBUTING.md) — Вклад в проект
- [SECURITY.md](SECURITY.md) — Безопасность

---

**Qwen-Claw Microservices** — AI-помощник с микросервисной архитектурой 🚀
