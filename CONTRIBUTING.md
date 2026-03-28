# Contributing to Qwen-Claw

Спасибо за интерес к проекту Qwen-Claw! Этот документ описывает процесс внесения изменений.

## 📋 Содержание

- [Code of Conduct](#code-of-conduct)
- [Как внести вклад](#как-внести-вклад)
- [Стандарты кода](#стандарты-кода)
- [Тестирование](#тестирование)
- [Pull Request процесс](#pull-request-процесс)
- [Release процесс](#release-процесс)

---

## Code of Conduct

- Будьте уважительны к другим участникам
- Конструктивная критика приветствуется
- Фокус на улучшении проекта

---

## Как внести вклад

### 1. Найти задачу

- Проверьте [Issues](https://github.com/sizotov81-hub/qwen-claw/issues)
- Ищите метки `good first issue`, `help wanted`
- Создайте свой issue если нашли баг

### 2. Fork и clone

```bash
git clone https://github.com/your-username/qwen-claw.git
cd qwen-claw
git remote add upstream https://github.com/sizotov81-hub/qwen-claw.git
```

### 3. Создать ветку

```bash
git checkout -b feature/your-feature
# или
git checkout -b fix/issue-123
```

### 4. Внести изменения

Следуйте [стандартам кода](#стандарты-кода).

### 5. Закоммитить

```bash
git add .
git commit -m "feat: add your feature"
```

### 6. Запушить

```bash
git push origin feature/your-feature
```

### 7. Создать Pull Request

- Откройте PR на GitHub
- Заполните описание
- Дождитесь review

---

## Стандарты кода

### Go стиль

- Следуйте [Effective Go](https://golang.org/doc/effective_go)
- Используйте `gofmt` или `goimports`
- Имена переменных: camelCase
- Имена функций/типов: PascalCase

### Линтинг

```bash
# Установить golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запустить
golangci-lint run ./...
```

### Структура кода

```go
// Правильно:
type UserService struct {
    db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
    return &UserService{db: db}
}

// Неправильно:
type userService struct {  // должно быть UserService
    DB *sql.DB  // должно быть db
}
```

### Обработка ошибок

```go
// Правильно:
if err != nil {
    return fmt.Errorf("create user: %w", err)
}

// Неправильно:
if err != nil {
    log.Println(err)  // игнорирование ошибки
}
```

### Логирование

```go
// Правильно:
logger.Infow("user created", "user_id", userID, "email", email)

// Неправильно:
fmt.Println("user created")
```

---

## Тестирование

### Запуск тестов

```bash
# Все тесты
./test-all.sh

# Unit тесты
go test ./...

# Конкретный пакет
go test ./internal/logger/...

# С покрытием
go test -coverprofile=coverage.out ./...
```

### Написание тестов

```go
func TestUserService_Create(t *testing.T) {
    // Arrange
    svc := NewUserService(mockDB)
    user := &User{Name: "test"}

    // Act
    err := svc.Create(user)

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

### Требования к тестам

- Покрытие > 80% для нового кода
- Table-driven тесты где применимо
- Mock внешних зависимостей
- Нет зависимостей между тестами

---

## Pull Request процесс

### Чеклист перед submit

- [ ] Код отформатирован (`gofmt`)
- [ ] Линтер проходит (`golangci-lint`)
- [ ] Все тесты проходят
- [ ] Покрытие не уменьшилось
- [ ] Документация обновлена
- [ ] CHANGELOG обновлён (если нужно)

### Описание PR

```markdown
## Описание
Краткое описание изменений.

## Тип изменений
- [ ] Bug fix (non-breaking change)
- [ ] New feature (non-breaking change)
- [ ] Breaking change
- [ ] Documentation update

## Тестирование
Опишите как тестировали.

## Чеклист
- [ ] Код отформатирован
- [ ] Линтер проходит
- [ ] Все тесты проходят
- [ ] Документация обновлена
```

### Review процесс

1. **Automated checks** — CI/CD pipeline
2. **Code review** — минимум 1 approval
3. **Merge** — squash merge в main

---

## Release процесс

### Версионирование

Следуем [Semantic Versioning](https://semver.org/):

- **MAJOR** — breaking changes
- **MINOR** — new features (backward compatible)
- **PATCH** — bug fixes

### Создание релиза

```bash
# Создать тег
git tag -a v1.0.0 -m "Release v1.0.0"

# Запушить тег
git push origin v1.0.0
```

GitHub Actions автоматически:
- Создаст GitHub Release
- Соберёт бинарники
- Запушит Docker образы

### CHANGELOG

Формат:

```markdown
## [1.0.0] - 2026-03-28

### Added
- Новая функциональность

### Changed
- Изменения в существующей функциональности

### Deprecated
- Устаревшая функциональность

### Removed
- Удалённая функциональность

### Fixed
- Исправления багов

### Security
- Исправления уязвимостей
```

---

## Вопросы?

- [GitHub Issues](https://github.com/sizotov81-hub/qwen-claw/issues)
- [Discussions](https://github.com/sizotov81-hub/qwen-claw/discussions)

---

**Спасибо за вклад!** 🎉
