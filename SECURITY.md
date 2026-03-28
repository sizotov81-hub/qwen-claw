# Security Policy

## Поддерживаемые версии

| Версия | Поддержка |
|--------|-----------|
| 1.0.x  | ✅        |
| 0.9.x  | ⚠️ (до 2026-06-01) |
| < 0.9  | ❌        |

---

## Сообщение об уязвимостях

**Пожалуйста, не создавайте Issues для уязвимостей!**

Отправляйте отчёты об уязвимостях на:
- Email: security@qwen-claw.dev (если настроено)
- GitHub Security Advisories: [Link](https://github.com/sizotov81-hub/qwen-claw/security/advisories)

### Что включать в отчёт

- Описание уязвимости
- Шаги для воспроизведения
- Версия(и) под воздействием
- Потенциальное влияние
- Возможное решение (если есть)

### Время ответа

- **Initial response:** в течение 48 часов
- **Status update:** в течение 5 дней
- **Fix:** зависит от сложности

---

## Безопасность в проекте

### Хранение секретов

**✅ ДЕЛАЙТЕ:**

```bash
# Храните секреты в .env (исключён из git)
QWEN_CLAW_TELEGRAM_TOKEN="your_token"

# Используйте Kubernetes Secrets
kubectl create secret generic qwen-claw-secrets \
  --from-literal=db-password='secret'

# Используйте GitHub Secrets для CI/CD
```

**❌ НЕ ДЕЛАЙТЕ:**

```bash
# Не коммитьте .env в git
# Не передавайте секреты в аргументах
# Не логируйте секреты
```

### Безопасность кода

#### Обработка ошибок

```go
// ✅ Правильно:
if err != nil {
    return fmt.Errorf("create user: %w", err)
}

// ❌ Неправильно:
if err != nil {
    log.Println(err)  // игнорирование
}
```

#### SQL инъекции

```go
// ✅ Правильно (prepared statements):
err := db.QueryRow("SELECT * FROM users WHERE id = $1", userID)

// ❌ Неправильно:
query := fmt.Sprintf("SELECT * FROM users WHERE id = %s", userID)
```

#### XSS защита

```go
// ✅ Правильно (html/template):
tmpl := template.Must(template.New("name").Parse(htmlTemplate))

// ❌ Неправильно:
fmt.Fprintf(w, "<div>%s</div>", userInput)  // нет экранирования
```

### Зависимости

Проект использует автоматическое сканирование зависимостей:

- **govulncheck** — при каждом PR
- **dependency-review** — проверка новых зависимостей
- **Trivy** — сканирование Docker образов

#### Обновление зависимостей

```bash
# Проверить устаревшие
go list -u -m all

# Обновить
go get -u ./...
go mod tidy
```

### Docker безопасность

```dockerfile
# ✅ Правильно:
FROM alpine:3.19
RUN addgroup -g 1000 qwenclaw && adduser -D -u 1000 -G qwenclaw qwenclaw
USER qwenclaw

# ❌ Неправильно:
FROM ubuntu:latest
USER root
```

### Kubernetes безопасность

```yaml
# ✅ Правильно:
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL

# ❌ Неправильно:
securityContext:
  privileged: true
  runAsUser: 0
```

---

## Security сканирование

### Автоматическое (GitHub Actions)

| Scan | Частота | Инструмент |
|------|---------|------------|
| Go vulnerabilities | Каждый PR | govulncheck |
| Dependency review | Каждый PR | dependency-review |
| Docker scan | Каждый push | Trivy |
| CodeQL analysis | Ежедневно | CodeQL |

### Локальное сканирование

```bash
# Go уязвимости
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Docker образ
docker scan qwen-claw/api-gateway:latest

# Зависимости
go list -m -json all | nancy sleuth
```

---

## Best Practices

### Для разработчиков

1. **Регулярно обновляйте зависимости**
2. **Используйте prepared statements для SQL**
3. **Экранируйте пользовательский ввод**
4. **Не логируйте секреты**
5. **Используйте HTTPS для внешних запросов**

### Для пользователей

1. **Меняйте пароли по умолчанию**
2. **Используйте HTTPS/TLS**
3. **Ограничивайте доступ к API**
4. **Регулярно обновляйте**
5. **Мониторьте логи на подозрительную активность**

---

## Incident Response Plan

### 1. Detection

- Автоматическое сканирование
- Отчёты от пользователей
- Security advisories

### 2. Assessment

- Оценка воздействия
- Определение затронутых версий
- Приоритизация

### 3. Containment

- Изоляция затронутых компонентов
- Временные workaround

### 4. Eradication

- Создание fix
- Тестирование fix

### 5. Recovery

- Деплой fix
- Мониторинг

### 6. Lessons Learned

- Post-mortem
- Обновление процессов

---

## Контакты

- **Security Team:** security@qwen-claw.dev
- **GitHub:** https://github.com/sizotov81-hub/qwen-claw/security

---

**Последнее обновление:** 2026-03-28
