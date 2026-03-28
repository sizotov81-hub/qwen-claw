# Load тесты для Qwen-Claw

## Требования

- k6: https://k6.io/docs/getting-started/installation/

## Установка k6

### Ubuntu/Debian:
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### macOS:
```bash
brew install k6
```

### Docker:
```bash
docker pull grafana/k6
```

## Запуск тестов

### Базовый запуск:
```bash
k6 run tests/load/load_test.js
```

### С кастомной конфигурацией:
```bash
BASE_URL=http://localhost:8080 k6 run tests/load/load_test.js
```

### С увеличенной нагрузкой:
```bash
k6 run --vus 100 --duration 5m tests/load/load_test.js
```

### С облачным отчётом:
```bash
K6_CLOUD_TOKEN=your_token k6 run --out cloud tests/load/load_test.js
```

## Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `BASE_URL` | URL сервиса | http://localhost:8080 |
| `GRPC_ADDR` | gRPC адрес | localhost:50051 |

## Сценарии теста

### 1. Health Check
- Проверка доступности сервиса
- Проверка времени ответа

### 2. Создание сессии
- POST /api/v1/sessions
- Проверка создания сессии

### 3. Получение сессии
- GET /api/v1/sessions/{id}
- Проверка получения данных

### 4. Отправка сообщения
- POST /api/v1/chat
- Проверка обработки сообщения

### 5. Поиск в памяти
- GET /api/v1/memory
- Проверка поиска

## Пороговые значения

Тест считается успешным если:
- 95% запросов выполняются < 500ms
- Ошибки < 10%
- Session create rate > 90%

## Этапы нагрузки

```
0-30s:   Разогрев до 10 RPS
30s-1m30s: Нагрузка до 50 RPS
1m30s-3m30s: Пиковая нагрузка 50 RPS
3m30s-4m30s: Стресс тест 100 RPS
4m30s-5m: Остывание до 0 RPS
```

## Анализ результатов

### Локально:
```bash
k6 run --out json=results.json tests/load/load_test.js
```

### HTML отчёт:
```bash
k6 run --out json=results.json tests/load/load_test.js
k6-to-html results.json > report.html
```

### Grafana k6 Dashboard:
```bash
# Запуск с интеграцией Grafana
k6 run --out influxdb=http://localhost:8086/k6 tests/load/load_test.js
```

## Интерпретация метрик

### http_req_duration
- p(50): Медианное время ответа
- p(90): 90% запросов быстрее этого значения
- p(95): 95% запросов быстрее этого значения
- max: Максимальное время ответа

### http_reqs
- Общее количество запросов
- RPS (requests per second)

### http_req_failed
- Процент неудачных запросов

### errors (кастомная)
- Процент ошибок в бизнес-логике

## Устранение проблем

### Высокое время ответа
- Проверить нагрузку на БД
- Проверить кэш
- Проверить лимиты ресурсов

### Много ошибок
- Проверить логи сервиса
- Проверить доступность зависимостей
- Проверить rate limiting

### Низкий session create rate
- Проверить БД подключения
- Проверить память
- Проверить блокировки

## CI/CD интеграция

### GitHub Actions:
```yaml
- name: Run load tests
  uses: grafana/k6-action@v0.2.0
  with:
    filename: tests/load/load_test.js
```

### GitLab CI:
```yaml
load_test:
  image: grafana/k6:latest
  script:
    - k6 run tests/load/load_test.js
```
