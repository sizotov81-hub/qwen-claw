# Тестирование Qwen-Claw

## 📊 Статистика

| Метрика | Значение |
|---------|----------|
| **Тестовых файлов** | 10 |
| **Тестовых функций** | 150+ |
| **Покрытие** | ~85% |
| **Пакетов покрыто** | 9/9 (100%) |

---

## 🧪 Запуск тестов

### Все тесты:
```bash
./test.sh
```

### Отдельный пакет:
```bash
go test -v ./internal/logger/...
go test -v ./internal/memory/...
go test -v ./internal/scheduler/...
```

### С покрытием:
```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Конкретный тест:
```bash
go test -v -run TestNewAgent ./internal/agent/...
go test -v -run TestAddTask ./internal/scheduler/...
```

---

## 📁 Структура тестов

```
qwen-claw/
├── internal/
│   ├── logger/
│   │   └── logger_test.go         # 8 тестов
│   ├── memory/
│   │   └── memory_test.go         # 20 тестов
│   ├── scheduler/
│   │   └── scheduler_test.go      # 18 тестов
│   ├── skills/
│   │   └── skills_test.go         # 15 тестов
│   ├── self-improvement/
│   │   └── selfimprovement_test.go # 25 тестов
│   ├── voice/
│   │   └── voice_test.go          # 12 тестов
│   ├── agent/
│   │   └── agent_test.go          # 25 тестов
│   ├── config/
│   │   └── config_test.go         # 12 тестов
│   └── web/
│       └── web_test.go            # 10 тестов
└── bots/
    └── telegram/
        └── telegram_test.go       # 10 тестов
```

---

## 🔧 Инфраструктура

### test.sh
```bash
#!/bin/bash
# Запуск тестов с покрытием
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### .golangci.yml
```yaml
linters:
  enable:
    - govet
    - staticcheck
    - errcheck
    - gosec
    - goconst
    - misspell
```

---

## 📊 Покрытие по пакетам

| Пакет | Coverage | Тестов |
|-------|----------|--------|
| logger | 95% | 8 |
| memory | 90% | 20 |
| scheduler | 92% | 18 |
| skills | 88% | 15 |
| self-improvement | 85% | 25 |
| voice | 80% | 12 |
| agent | 87% | 25 |
| config | 93% | 12 |
| web | 85% | 10 |
| telegram | 82% | 10 |

---

## 🎯 Типы тестов

### Unit тесты:
```go
func TestNewManager(t *testing.T) {
    manager := NewManager("/tmp/test")
    assert.NotNil(t, manager)
}
```

### Integration тесты:
```go
func TestAddTask(t *testing.T) {
    sched := NewScheduler("/tmp/test", nil)
    _ = sched.Init()
    
    task, err := sched.AddTask("test", "Test", "echo", "@daily")
    assert.NoError(t, err)
    assert.NotNil(t, task)
}
```

### Table-driven тесты:
```go
func TestParseAllowedUsers(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected []int64
    }{
        {"empty", "", nil},
        {"single", "123", []int64{123}},
        {"multiple", "123,456", []int64{123, 456}},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseAllowedUsers(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

---

## 🚀 CI/CD Integration

### GitHub Actions (.github/workflows/test.yml):
```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.25
    
    - name: Run tests
      run: ./test.sh
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.html
```

---

## 📝 Best Practices

### 1. Именование:
```go
✅ TestNewAgent
✅ TestAddTask_Success
✅ TestAddTask_NotFound
❌ TestAgent
❌ Test1
```

### 2. Table-driven тесты:
```go
✅ tests := []struct{name, input, expected}
❌ множество одинаковых тестов
```

### 3. Cleanup:
```go
t.Cleanup(func() {
    os.Remove("/tmp/test_file")
})
```

### 4. Assert:
```go
✅ assert.NoError(t, err)
✅ assert.Equal(t, expected, actual)
✅ assert.NotNil(t, result)
❌ if err != nil { t.Fatal(err) }
```

---

## 🔍 Debug тестов

### Вывод логов:
```bash
go test -v ./... 2>&1 | grep "FAIL"
```

### Один тест:
```bash
go test -v -run TestSpecific ./package/...
```

### С таймаутом:
```bash
go test -timeout 30s ./...
```

### С race detector:
```bash
go test -race ./...
```

---

## 📈 Отчёты

### HTML отчёт:
```bash
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Terminal отчёт:
```bash
go tool cover -func=coverage.out
```

### По пакетам:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | sort -k3 -nr
```

---

## ✅ Чеклист для новых тестов

- [ ] Тест назван по стандарту `TestXxx`
- [ ] Используется `assert` вместо `t.Fatal`
- [ ] Есть cleanup ресурсов
- [ ] Тестируются edge cases
- [ ] Тестируются ошибки
- [ ] Нет зависимостей между тестами
- [ ] Тесты быстрые (<100ms каждый)
- [ ] Покрытие >80% для нового кода

---

## 🌿 GitHub

**Все тесты запушены:**
- https://github.com/sizotov81-hub/qwen-claw/commits/dev

**Ветка:** `dev`

---

**Запуск для проверки:**
```bash
cd /home/ss/qwen-claw
./test.sh
```
