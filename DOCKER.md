# Qwen-Claw в Docker

Этот документ описывает запуск Qwen-Claw в Docker контейнере.

## Информация об образе

- **Размер:** ~104 MB
- **Базовый образ:** Alpine 3.19
- **Go версия:** 1.21
- **Пользователь:** qwenclaw (UID 1000)

**Примечание:** Qwen Code CLI может быть установлен в контейнере автоматически или предоставлен через volume. Для полной функциональности рекомендуется запускать с хост-установленным Qwen Code CLI.

## Быстрый старт

### 1. Клонирование репозитория

```bash
cd /home/ss/qwen-claw
```

### 2. Настройка конфигурации

Скопируйте файлы конфигурации:

```bash
cp config.yaml.example config.yaml
cp .env.example .env
```

Отредактируйте `.env` и добавьте ваш Telegram токен:

```bash
nano .env
```

```env
QWEN_CLAW_TELEGRAM_TOKEN="your-bot-token-here"
QWEN_CLAW_TELEGRAM_ALLOWED_USERS="your-telegram-id"
```

### 3. Запуск с docker-compose

```bash
# Сборка и запуск
docker-compose up -d

# Просмотр логов
docker-compose logs -f

# Остановка
docker-compose down
```

### 4. Запуск без docker-compose

```bash
# Сборка образа
docker build -t qwen-claw:latest .

# Запуск контейнера
docker run -d \
  --name qwen-claw \
  -e QWEN_CLAW_TELEGRAM_TOKEN="your-token" \
  -e QWEN_CLAW_TELEGRAM_ALLOWED_USERS="your-id" \
  -v qwen-claw-data:/app/.qwen \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -v $(pwd)/.env:/app/.env:ro \
  --restart unless-stopped \
  qwen-claw:latest telegram
```

## Команды запуска

По умолчанию контейнер запускает Telegram бота. Вы можете изменить команду:

```yaml
# docker-compose.yml
command: ["telegram"]  # Telegram бот
# command: ["run", "Привет!"]  # Одиночный запрос
# command: ["chat"]  # Интерактивный чат
# command: ["doctor"]  # Проверка установки
```

## Тома (Volumes)

| Том | Назначение |
|-----|------------|
| `qwen-claw-data` | Данные: память, задачи, история |
| `./config.yaml` | Конфигурация (read-only) |
| `./.env` | Переменные окружения (read-only) |
| `./logs/` | Логи (опционально) |

## Переменные окружения

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `QWEN_CLAW_TELEGRAM_TOKEN` | Токен Telegram бота | - |
| `QWEN_CLAW_TELEGRAM_ALLOWED_USERS` | Разрешённые пользователи (ID через запятую) | - |
| `QWEN_CLAW_MODEL` | Модель Qwen | (по умолчанию) |
| `QWEN_CLAW_APPROVAL_MODE` | Режим подтверждения | `auto-edit` |
| `QWEN_CLAW_DEBUG` | Режим отладки | `false` |
| `TZ` | Часовой пояс | `UTC` |

## Полезные команды

```bash
# Просмотр логов
docker-compose logs -f

# Перезапуск
docker-compose restart

# Остановка
docker-compose down

# Удаление всего (включая данные)
docker-compose down -v

# Выполнение команды в контейнере
docker-compose exec qwen-claw /app/qwen-claw doctor
docker-compose exec qwen-claw /app/qwen-claw skills
docker-compose exec qwen-claw /app/qwen-claw tasks

# Доступ к shell контейнера
docker-compose exec qwen-claw sh
```

## Мониторинг

```bash
# Статус контейнера
docker-compose ps

# Использование ресурсов
docker stats qwen-claw

# Проверка health
docker inspect --format='{{.State.Health.Status}}' qwen-claw
```

## Обновление

```bash
# Пересборка образа
docker-compose build --no-cache

# Обновление с сохранением данных
docker-compose pull
docker-compose up -d
```

## Резервное копирование

```bash
# Экспорт данных
docker run --rm \
  -v qwen-claw-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/qwen-claw-backup.tar.gz /data

# Импорт данных
docker run --rm \
  -v qwen-claw-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar xzf /backup/qwen-claw-backup.tar.gz -C /
```

## Отладка

```bash
# Запуск в интерактивном режиме
docker-compose run --rm qwen-claw -i

# Запуск с подробным выводом
docker-compose run --rm qwen-claw --verbose run "тест"

# Проверка конфигурации
docker-compose run --rm qwen-claw doctor
```

## Проблемы и решения

### Контейнер не запускается

Проверьте логи:
```bash
docker-compose logs
```

### Telegram бот не отвечает

1. Проверьте токен в `.env`
2. Убедитесь, что пользователь добавлен в `ALLOWED_USERS`
3. Проверьте логи бота

### Ошибка "Qwen Code CLI not found"

Qwen Code CLI устанавливается при сборке образа. Если проблема сохраняется, пересоберите образ:
```bash
docker-compose build --no-cache
```

### Потеря данных

Данные хранятся в томе `qwen-claw-data`. Для сохранения используйте резервное копирование.

## Безопасность

- Контейнер запускается от непривилегированного пользователя (`qwenclaw:1000`)
- Конфигурационные файлы монтируются как read-only
- Не храните секреты в `config.yaml` — используйте `.env`

## Ресурсы

По умолчанию контейнеру выделено:
- CPU: 0.25 - 1.0 ядра
- RAM: 128M - 512M

Настройте в `docker-compose.yml` при необходимости.
