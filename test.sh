#!/bin/bash
# Запуск тестов с покрытием

set -e

echo "🧪 Запуск тестов Qwen-Claw..."
echo ""

# Запуск тестов с покрытием
go test -v -race -coverprofile=coverage.out ./... 2>&1 | tee test_output.txt

# Генерация HTML отчёта
go tool cover -html=coverage.out -o coverage.html

# Вывод итогового покрытия
echo ""
echo "📊 Итоговое покрытие:"
go tool cover -func=coverage.out | tail -1

# Очистка
rm -f coverage.out

echo ""
echo "✅ Тесты завершены!"
echo "📄 HTML отчёт: coverage.html"
echo "📝 Лог тестов: test_output.txt"
