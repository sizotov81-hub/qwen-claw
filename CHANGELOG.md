# Changelog

Все значимые изменения в проекте Qwen-Claw.

Формат следует [Keep a Changelog](https://keepachangelog.com/ru/1.0.0/),
версионирование — [Semantic Versioning](https://semver.org/lang/ru/).

---

## [1.0.0] - 2026-03-28

### Добавлено

#### Этап 1: Инфраструктура
- Docker Compose для локальной разработки
- Базовые Dockerfile для сервисов
- PostgreSQL + pgVector инициализация
- Prometheus и Grafana конфигурация
- Jaeger для трассировки

#### Этап 2: Базовая инфраструктура
- Protocol Buffers определения (5 сервисов)
- Пакет конфигурации с Viper
- Базовый gRPC сервер
- 17 unit тестов (100% покрытие)

#### Этап 3: Микросервисы
- Session & Memory Service (модели, хранилище, кэш)
- gRPC сервер для Session Memory
- 24 unit теста (100% покрытие)

#### Этап 4: Межсервисное взаимодействие
- gRPC клиенты для всех сервисов
- RabbitMQ клиент для асинхронных задач
- Circuit Breaker реализация
- 37 unit тестов (100% покрытие)

#### Этап 5: Наблюдаемость
- Расширенный Zap логгер
- Prometheus метрики (HTTP, gRPC, БД, кэш, RabbitMQ, CB)
- 14 unit тестов (100% покрытие)

#### Этап 6: Тестирование
- Integration тесты (11 тестов)
- Load тесты с k6 (5 сценариев)
- test-all.sh скрипт
- Общее покрытие 94.3%

#### Этап 7: CI/CD и деплой
- GitHub Actions workflows (CI/CD, Security, Release)
- Kubernetes манифесты (6 файлов)
- Helm chart с dependencies
- Docker образы для 5 сервисов

#### Этап 8: Документация
- README.md с полной документацией
- CONTRIBUTING.md
- MICROSERVICES_ARCHITECTURE.md
- STAGE_*_COMPLETE.md отчёты

#### Этап 9: Система навыков
- Формат манифеста SKILL.md с YAML frontmatter
- CLI для управления навыками (list, run, info, validate, audit)
- Песочница для навыков (Docker isolation)
- Система разрешений (network, filesystem, elevated)
- Аудит выполнения (JSONL логирование)
- Локальный реестр навыков
- Встроенные навыки (7): shell, file, search, memory, git, http, notify
- Примеры навыков (3): web-search, code-review, github-tools
- Документация (skills/README.md, skills/GUIDE.md)

### Изменено

- Рефакторинг логирования (zap вместо log)
- Улучшена обработка ошибок (%w для обёртывания)
- Оптимизирована работа с памятью (Inverted Index)
- Улучшена конфигурация (кэширование)

### Исправлено

- Тесты Telegram бота (4 failing теста)
- skills/developer компиляция
- Deadlock в ContextManager
- Утечки памяти в сессиях

### Безопасность

- Добавлен security scanning (govulncheck, CodeQL, Trivy)
- Secrets вынесены в Kubernetes Secrets
- Non-root пользователи в Docker
- Rate limiting для всех сервисов

---

## [0.9.0] - 2026-03-21

### Добавлено

- Базовая версия Qwen-Claw
- Telegram бот интеграция
- Планировщик задач
- Система памяти (4 уровня)
- Web UI (базовый)

### Изменено

- Рефакторинг agent package
- Улучшена обработка контекста

### Исправлено

- Обработка ошибок в scheduler
- Логирование (log → logger)

---

## [0.1.0] - 2026-03-01

### Добавлено

- Первый релиз
- Базовый CLI
- Интеграция с Qwen Code CLI

---

## Типы изменений

- **Added** — новые функции
- **Changed** — изменения в существующей функциональности
- **Deprecated** — устаревшая функциональность
- **Removed** — удалённая функциональность
- **Fixed** — исправления багов
- **Security** — исправления уязвимостей

---

**Ссылки:**

- [1.0.0]: https://github.com/sizotov81-hub/qwen-claw/releases/tag/v1.0.0
- [0.9.0]: https://github.com/sizotov81-hub/qwen-claw/releases/tag/v0.9.0
- [0.1.0]: https://github.com/sizotov81-hub/qwen-claw/releases/tag/v0.1.0
