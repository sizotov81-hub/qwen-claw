#!/bin/bash
# Скрипт установки Qwen-Claw в PATH

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$HOME/bin"

echo "🔧 Установка Qwen-Claw..."

# Создаём директорию bin если не существует
if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
    echo "✓ Создана директория $INSTALL_DIR"
fi

# Собираем бинарник
echo "📦 Сборка бинарника..."
cd "$SCRIPT_DIR"
CGO_ENABLED=0 go build -o qwen-claw ./cmd/main.go
echo "✓ Бинарник собран"

# Копируем в PATH
cp qwen-claw "$INSTALL_DIR/"
echo "✓ Бинарник скопирован в $INSTALL_DIR/qwen-claw"

# Проверяем PATH
if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
    echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
    echo "✓ Добавлено в ~/.bashrc"
    echo "  Для применения выполните: source ~/.bashrc"
fi

# Проверяем установку
if command -v qwen-claw &> /dev/null; then
    echo ""
    echo "✅ Установка завершена!"
    echo ""
    echo "Теперь можно запускать qwen-claw из любой директории:"
    echo "  qwen-claw doctor"
    echo "  qwen-claw telegram"
    echo "  qwen-claw run \"ваш запрос\""
else
    echo ""
    echo "⚠️  qwen-claw не найден в PATH"
    echo "  Выполните: source ~/.bashrc"
    echo "  Или добавьте ~/bin в PATH вручную"
fi
