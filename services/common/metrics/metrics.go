// Package metrics предоставляет Prometheus метрики для микросервисов Qwen-Claw
package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Metrics менеджер метрик
type Metrics struct {
	mu sync.RWMutex

	// Request метрики
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	requestInFlight prometheus.Gauge

	// gRPC метрики
	grpcRequestsTotal   *prometheus.CounterVec
	grpcRequestDuration *prometheus.HistogramVec

	// Database метрики
	dbConnections      prometheus.Gauge
	dbQueriesTotal     *prometheus.CounterVec
	dbQueryDuration    *prometheus.HistogramVec
	dbConnectionsInUse prometheus.Gauge

	// Cache метрики
	cacheHits   *prometheus.CounterVec
	cacheMisses *prometheus.CounterVec
	cacheSize   prometheus.Gauge

	// RabbitMQ метрики
	mqMessagesPublished *prometheus.CounterVec
	mqMessagesConsumed  *prometheus.CounterVec
	mqQueueLength       *prometheus.GaugeVec

	// Circuit Breaker метрики
	cbState *prometheus.GaugeVec

	// Registry
	registry *prometheus.Registry

	logger *zap.SugaredLogger
}

// Config конфигурация метрик
type Config struct {
	ServiceName string
	Port        int
	Path        string
	Enabled     bool
	Logger      *zap.SugaredLogger
}

// NewMetrics создаёт новый менеджер метрик
func NewMetrics(cfg *Config) (*Metrics, error) {
	if !cfg.Enabled {
		return &Metrics{}, nil
	}

	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop().Sugar()
	}

	m := &Metrics{
		registry: prometheus.NewRegistry(),
		logger:   cfg.Logger,
	}

	// Регистрируем стандартные метрики Go
	m.registry.MustRegister(prometheus.NewGoCollector())
	m.registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	// Request метрики
	m.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"method", "endpoint", "status"},
	)
	m.registry.MustRegister(m.requestsTotal)

	m.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request duration in seconds",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
			Buckets:     prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	m.registry.MustRegister(m.requestDuration)

	m.requestInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "http_requests_in_flight",
			Help:        "Number of HTTP requests currently being processed",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
	)
	m.registry.MustRegister(m.requestInFlight)

	// gRPC метрики
	m.grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "grpc_requests_total",
			Help:        "Total number of gRPC requests",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"method", "status"},
	)
	m.registry.MustRegister(m.grpcRequestsTotal)

	m.grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "grpc_request_duration_seconds",
			Help:        "gRPC request duration in seconds",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
			Buckets:     prometheus.DefBuckets,
		},
		[]string{"method"},
	)
	m.registry.MustRegister(m.grpcRequestDuration)

	// Database метрики
	m.dbConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "database_connections",
			Help:        "Number of database connections",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
	)
	m.registry.MustRegister(m.dbConnections)

	m.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "database_queries_total",
			Help:        "Total number of database queries",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"query_type"},
	)
	m.registry.MustRegister(m.dbQueriesTotal)

	m.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "database_query_duration_seconds",
			Help:        "Database query duration in seconds",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
			Buckets:     prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)
	m.registry.MustRegister(m.dbQueryDuration)

	m.dbConnectionsInUse = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "database_connections_in_use",
			Help:        "Number of database connections currently in use",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
	)
	m.registry.MustRegister(m.dbConnectionsInUse)

	// Cache метрики
	m.cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "cache_hits_total",
			Help:        "Total number of cache hits",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"cache_type"},
	)
	m.registry.MustRegister(m.cacheHits)

	m.cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "cache_misses_total",
			Help:        "Total number of cache misses",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"cache_type"},
	)
	m.registry.MustRegister(m.cacheMisses)

	m.cacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "cache_size",
			Help:        "Current cache size",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
	)
	m.registry.MustRegister(m.cacheSize)

	// RabbitMQ метрики
	m.mqMessagesPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "rabbitmq_messages_published_total",
			Help:        "Total number of messages published to RabbitMQ",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"exchange", "queue"},
	)
	m.registry.MustRegister(m.mqMessagesPublished)

	m.mqMessagesConsumed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "rabbitmq_messages_consumed_total",
			Help:        "Total number of messages consumed from RabbitMQ",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"queue"},
	)
	m.registry.MustRegister(m.mqMessagesConsumed)

	m.mqQueueLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "rabbitmq_queue_length",
			Help:        "Current length of RabbitMQ queues",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"queue"},
	)
	m.registry.MustRegister(m.mqQueueLength)

	// Circuit Breaker метрики
	m.cbState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        "circuit_breaker_state",
			Help:        "Current state of circuit breakers (0=closed, 1=open, 2=half-open)",
			ConstLabels: prometheus.Labels{"service": cfg.ServiceName},
		},
		[]string{"breaker_name"},
	)
	m.registry.MustRegister(m.cbState)

	// Запускаем HTTP сервер для метрик
	go m.startHTTPServer(cfg.Port, cfg.Path)

	m.logger.Infow("Metrics initialized", "port", cfg.Port, "path", cfg.Path)

	return m, nil
}

// startHTTPServer запускает HTTP сервер для метрик
func (m *Metrics) startHTTPServer(port int, path string) {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	addr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	m.logger.Infow("Metrics server starting", "addr", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		m.logger.Errorw("Metrics server error", "error", err)
	}
}

// HTTP метрики

// IncRequestsTotal увеличивает счётчик запросов
func (m *Metrics) IncRequestsTotal(method, endpoint, status string) {
	if m.requestsTotal == nil {
		return
	}
	m.requestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// ObserveRequestDuration наблюдает длительность запроса
func (m *Metrics) ObserveRequestDuration(method, endpoint string, duration time.Duration) {
	if m.requestDuration == nil {
		return
	}
	m.requestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// IncRequestInFlight увеличивает счётчик активных запросов
func (m *Metrics) IncRequestInFlight() {
	if m.requestInFlight == nil {
		return
	}
	m.requestInFlight.Inc()
}

// DecRequestInFlight уменьшает счётчик активных запросов
func (m *Metrics) DecRequestInFlight() {
	if m.requestInFlight == nil {
		return
	}
	m.requestInFlight.Dec()
}

// gRPC метрики

// IncGRPCRequestsTotal увеличивает счётчик gRPC запросов
func (m *Metrics) IncGRPCRequestsTotal(method, status string) {
	if m.grpcRequestsTotal == nil {
		return
	}
	m.grpcRequestsTotal.WithLabelValues(method, status).Inc()
}

// ObserveGRPCRequestDuration наблюдает длительность gRPC запроса
func (m *Metrics) ObserveGRPCRequestDuration(method string, duration time.Duration) {
	if m.grpcRequestDuration == nil {
		return
	}
	m.grpcRequestDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// Database метрики

// SetDBConnections устанавливает количество подключений к БД
func (m *Metrics) SetDBConnections(count int) {
	if m.dbConnections == nil {
		return
	}
	m.dbConnections.Set(float64(count))
}

// IncDBQueriesTotal увеличивает счётчик запросов к БД
func (m *Metrics) IncDBQueriesTotal(queryType string) {
	if m.dbQueriesTotal == nil {
		return
	}
	m.dbQueriesTotal.WithLabelValues(queryType).Inc()
}

// ObserveDBQueryDuration наблюдает длительность запроса к БД
func (m *Metrics) ObserveDBQueryDuration(queryType string, duration time.Duration) {
	if m.dbQueryDuration == nil {
		return
	}
	m.dbQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

// SetDBConnectionsInUse устанавливает количество используемых подключений
func (m *Metrics) SetDBConnectionsInUse(count int) {
	if m.dbConnectionsInUse == nil {
		return
	}
	m.dbConnectionsInUse.Set(float64(count))
}

// Cache метрики

// IncCacheHits увеличивает счётчик попаданий в кэш
func (m *Metrics) IncCacheHits(cacheType string) {
	if m.cacheHits == nil {
		return
	}
	m.cacheHits.WithLabelValues(cacheType).Inc()
}

// IncCacheMisses увеличивает счётчик промахов кэша
func (m *Metrics) IncCacheMisses(cacheType string) {
	if m.cacheMisses == nil {
		return
	}
	m.cacheMisses.WithLabelValues(cacheType).Inc()
}

// SetCacheSize устанавливает размер кэша
func (m *Metrics) SetCacheSize(size int) {
	if m.cacheSize == nil {
		return
	}
	m.cacheSize.Set(float64(size))
}

// RabbitMQ метрики

// IncMQMessagesPublished увеличивает счётчик опубликованных сообщений
func (m *Metrics) IncMQMessagesPublished(exchange, queue string) {
	if m.mqMessagesPublished == nil {
		return
	}
	m.mqMessagesPublished.WithLabelValues(exchange, queue).Inc()
}

// IncMQMessagesConsumed увеличивает счётчик потреблённых сообщений
func (m *Metrics) IncMQMessagesConsumed(queue string) {
	if m.mqMessagesConsumed == nil {
		return
	}
	m.mqMessagesConsumed.WithLabelValues(queue).Inc()
}

// SetMQQueueLength устанавливает длину очереди
func (m *Metrics) SetMQQueueLength(queue string, length int) {
	if m.mqQueueLength == nil {
		return
	}
	m.mqQueueLength.WithLabelValues(queue).Set(float64(length))
}

// Circuit Breaker метрики

// SetCBState устанавливает состояние Circuit Breaker
func (m *Metrics) SetCBState(breakerName string, state int) {
	if m.cbState == nil {
		return
	}
	m.cbState.WithLabelValues(breakerName).Set(float64(state))
}

// Middleware создаёт HTTP middleware для метрик
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		m.IncRequestInFlight()
		defer m.DecRequestInFlight()

		// Обёртка для ResponseWriter чтобы получить статус код
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		m.ObserveRequestDuration(r.Method, r.URL.Path, duration)
		m.IncRequestsTotal(r.Method, r.URL.Path, fmt.Sprintf("%d", wrapped.status))
	})
}

// responseWriter обёртка для получения статуса ответа
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Close закрывает менеджер метрик
func (m *Metrics) Close() error {
	m.logger.Info("Metrics closed")
	return nil
}
