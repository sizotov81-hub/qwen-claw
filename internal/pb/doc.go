// Package proto содержит заглушки для сгенерированных Protocol Buffers
// В продакшене используйте: ./scripts/generate-proto.sh

package proto

//go:generate sh ../scripts/generate-proto.sh

// Примечание: После установки protoc и запуска generate-proto.sh
// в этой директории появятся файлы:
// - api_gateway.pb.go
// - api_gateway_grpc.pb.go
// - session_memory.pb.go
// - session_memory_grpc.pb.go
// - qwen_wrapper.pb.go
// - qwen_wrapper_grpc.pb.go
// - llm_proxy.pb.go
// - llm_proxy_grpc.pb.go
// - tools_executor.pb.go
// - tools_executor_grpc.pb.go
