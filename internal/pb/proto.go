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

// SessionMemoryService интерфейс для Session Memory Service
// В продакшене будет сгенерирован из session_memory.proto
type SessionMemoryService interface {
	// CreateSession создаёт сессию
	// CreateSession(context.Context, *CreateSessionRequest) (*Session, error)
	
	// GetSession получает сессию
	// GetSession(context.Context, *SessionID) (*Session, error)
	
	// AddMessage добавляет сообщение
	// AddMessage(context.Context, *AddMessageRequest) (*Message, error)
	
	// GetMessages получает сообщения
	// GetMessages(context.Context, *GetMessagesRequest) (*MessageList, error)
	
	// AddMemory добавляет запись памяти
	// AddMemory(context.Context, *MemoryEntry) (*MemoryEntry, error)
	
	// SearchMemory ищет в памяти
	// SearchMemory(context.Context, *SearchMemoryRequest) (*MemoryList, error)
}
