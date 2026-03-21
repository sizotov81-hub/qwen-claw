#!/bin/bash

# Тестирование планировщика, памяти и поиска Qwen-Claw
# Использование: ./test_realtime.sh

set -e

echo "🧪 Тестирование Qwen-Claw в реальном времени"
echo "=============================================="
echo ""

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Директория для тестов
TEST_DIR="/tmp/qwen_claw_test_$$"
SCHEDULER_DIR="$TEST_DIR/scheduler"
MEMORY_DIR="$TEST_DIR/memory"

# Создаём тестовые директории
mkdir -p "$SCHEDULER_DIR" "$MEMORY_DIR"

echo -e "${BLUE}📁 Тестовая директория:${NC} $TEST_DIR"
echo ""

# ============================================
# ТЕСТ 1: Проверка памяти
# ============================================
echo -e "${YELLOW}═══ ТЕСТ 1: Проверка памяти ═══${NC}"
echo ""

echo -e "${BLUE}1.1 Добавляем запись в память...${NC}"
go run cmd/main.go memory add "Тестовая запись: планировщик запущен в $(date)"
echo -e "${GREEN}✓ Запись добавлена${NC}"
echo ""

echo -e "${BLUE}1.2 Показываем память...${NC}"
go run cmd/main.go memory
echo ""

echo -e "${BLUE}1.3 Поиск по памяти...${NC}"
go run cmd/main.go memory search "планировщик"
echo ""

# ============================================
# ТЕСТ 2: Проверка планировщика
# ============================================
echo -e "${YELLOW}═══ ТЕСТ 2: Проверка планировщика ═══${NC}"
echo ""

echo -e "${BLUE}2.1 Добавляем задачу на 5 минут...${NC}"
echo "Задача будет выполнена через 5 минут после запуска"
echo ""

# Добавляем задачу с cron-выражением на 5 минут от текущего времени
CURRENT_MINUTE=$(date +%M)
CURRENT_HOUR=$(date +%H)
FUTURE_MINUTE=$(( (CURRENT_MINUTE + 5) % 60 ))
FUTURE_HOUR=$CURRENT_HOUR

# Если переход через час
if [ $FUTURE_MINUTE -lt $CURRENT_MINUTE ]; then
    FUTURE_HOUR=$(( (CURRENT_HOUR + 1) % 24 ))
fi

CRON_EXPR="$FUTURE_MINUTE $FUTURE_HOUR * * *"

echo -e "${BLUE}Cron выражение:${NC} $CRON_EXPR (выполнение в $FUTURE_HOUR:$FUTURE_MINUTE)"
go run cmd/main.go tasks add "test_5min_task" "$CRON_EXPR" "echo 'Задача выполнена через 5 минут!'"
echo ""

echo -e "${BLUE}2.2 Показываем список задач...${NC}"
go run cmd/main.go tasks
echo ""

echo -e "${BLUE}2.3 Выполняем задачу немедленно для проверки...${NC}"
# Получаем ID задачи (первую из списка)
TASK_ID=$(go run cmd/main.go tasks 2>/dev/null | grep "ID:" | head -1 | awk '{print $2}')

if [ -n "$TASK_ID" ]; then
    echo -e "${BLUE}ID задачи:${NC} $TASK_ID"
    go run cmd/main.go tasks run "$TASK_ID"
    echo -e "${GREEN}✓ Задача выполнена${NC}"
else
    echo -e "${RED}✗ Не удалось получить ID задачи${NC}"
fi
echo ""

echo -e "${BLUE}2.4 Показываем результаты выполнения...${NC}"
go run cmd/main.go tasks results
echo ""

# ============================================
# ТЕСТ 3: Проверка поиска в реальном времени
# ============================================
echo -e "${YELLOW}═══ ТЕСТ 3: Проверка поиска в реальном времени ═══${NC}"
echo ""

echo -e "${BLUE}3.1 Добавляем несколько записей...${NC}"
go run cmd/main.go memory add "Git: основная ветка называется main"
go run cmd/main.go memory add "Docker: порт приложения 8080"
go run cmd/main.go memory add "Node.js: версия 20.x"
echo -e "${GREEN}✓ Записи добавлены${NC}"
echo ""

echo -e "${BLUE}3.2 Поиск по ключевому слову 'Git'...${NC}"
go run cmd/main.go memory search "Git"
echo ""

echo -e "${BLUE}3.3 Поиск по ключевому слову 'Docker'...${NC}"
go run cmd/main.go memory search "Docker"
echo ""

echo -e "${BLUE}3.4 Поиск по ключевому слову 'Node'...${NC}"
go run cmd/main.go memory search "Node"
echo ""

# ============================================
# ТЕСТ 4: Проверка контекста
# ============================================
echo -e "${YELLOW}═══ ТЕСТ 4: Проверка контекста ═══${NC}"
echo ""

echo -e "${BLUE}4.1 Запускаем чат с контекстом...${NC}"
echo "Привет! Запомни: мой проект называется qwen-claw" | go run cmd/main.go run -
echo ""

echo -e "${BLUE}4.2 Проверяем контекст...${NC}"
go run cmd/main.go run "Что я просил запомнить?"
echo ""

# ============================================
# Очистка
# ============================================
echo -e "${YELLOW}═══ Очистка ═══${NC}"
echo ""
echo -e "${BLUE}Очищаем память...${NC}"
go run cmd/main.go memory clear
echo ""

echo -e "${BLUE}Удаляем тестовые файлы...${NC}"
rm -rf "$TEST_DIR"
echo -e "${GREEN}✓ Тестовые файлы удалены${NC}"
echo ""

echo -e "${GREEN}═══════════════════════════════════${NC}"
echo -e "${GREEN}✓ Все тесты завершены!${NC}"
echo -e "${GREEN}═══════════════════════════════════${NC}"
