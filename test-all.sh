#!/bin/bash
# Скрипт для запуска всех тестов Qwen-Claw

set -e

echo "🧪 Запуск тестов Qwen-Claw"
echo "=========================="
echo ""

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Счётчики
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Функция для запуска тестов пакета
run_tests() {
    local package=$1
    local description=$2
    
    echo -e "${YELLOW}Тестирование: ${description}${NC}"
    echo "Пакет: $package"
    
    # Запуск тестов с покрытием
    if CGO_ENABLED=0 go test -v -coverprofile=coverage.out "$package" -timeout 60s 2>&1; then
        echo -e "${GREEN}✅ Успешно${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ Не успешно${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
}

# Функция для запуска integration тестов
run_integration_tests() {
    local package=$1
    local description=$2
    
    echo -e "${YELLOW}Integration тесты: ${description}${NC}"
    echo "Пакет: $package"
    
    if CGO_ENABLED=0 go test -v "$package" -timeout 120s 2>&1; then
        echo -e "${GREEN}✅ Успешно${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}❌ Не успешно${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo ""
}

# Основная секция
echo "📦 Unit тесты"
echo "-------------"

run_tests "./internal/logger/..." "Logger"
run_tests "./internal/metrics/..." "Metrics"
run_tests "./internal/session/..." "Session Models"
run_tests "./internal/session/service/..." "Session Service"
run_tests "./internal/session/grpc/..." "Session gRPC"
run_tests "./internal/client/..." "gRPC Clients"
run_tests "./internal/rabbitmq/..." "RabbitMQ"
run_tests "./internal/circuitbreaker/..." "Circuit Breaker"
run_tests "./internal/server/..." "gRPC Server"
run_tests "./pkg/config/..." "Config"

echo ""
echo "📦 Integration тесты"
echo "--------------------"

run_integration_tests "./tests/integration/..." "Integration Tests"

echo ""
echo "📊 Покрытие"
echo "-----------"

# Объединение coverage профилей
if [ -f coverage.out ]; then
    echo "Генерация HTML отчёта..."
    go tool cover -html=coverage.out -o coverage.html
    echo "HTML отчёт: coverage.html"
    
    # Статистика покрытия
    echo ""
    echo "Статистика покрытия:"
    go tool cover -func=coverage.out | tail -1
fi

echo ""
echo "=========================="
echo "📈 Итоги"
echo "=========================="
echo "Всего тестов: $TOTAL_TESTS"
echo -e "${GREEN}Пройдено: $PASSED_TESTS${NC}"
echo -e "${RED}Не пройдено: $FAILED_TESTS${NC}"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ Все тесты пройдены!${NC}"
    exit 0
else
    echo -e "${RED}❌ Некоторые тесты не пройдены${NC}"
    exit 1
fi
