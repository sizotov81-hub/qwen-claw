#!/bin/bash
# Скрипт для генерации Go кода из Protocol Buffers

set -e

echo "🔧 Генерация Go кода из Protocol Buffers..."

# Проверяем наличие protoc
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc не найден. Установите protobuf-compiler"
    exit 1
fi

# Проверяем наличие плагинов Go
if ! command -v protoc-gen-go &> /dev/null; then
    echo "📦 Устанавливаем protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "📦 Устанавливаем protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Директории
PROTO_DIR="pkg/api/proto"
OUTPUT_DIR="internal/pb"

# Создаём директорию вывода
mkdir -p "$OUTPUT_DIR"

# Генерируем код для каждого .proto файла
for proto_file in "$PROTO_DIR"/*.proto; do
    if [ -f "$proto_file" ]; then
        echo "📝 Генерация для $(basename $proto_file)..."
        protoc \
            --go_out="$OUTPUT_DIR" \
            --go_opt=paths=source_relative \
            --go-grpc_out="$OUTPUT_DIR" \
            --go-grpc_opt=paths=source_relative \
            "$proto_file"
    fi
done

echo "✅ Генерация завершена!"
echo "📁 Файлы сохранены в $OUTPUT_DIR"
