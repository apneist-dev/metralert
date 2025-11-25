package metrics

type Gauge float64
type Counter int64

// generate:reset
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

// generate:reset
type MetricsGroup struct {
	Slice []Metrics
}

// generate:reset
type AuditMetrics struct {
	TS          int64    `json:"ts"`
	MetricNames []string `json:"metrics"`
	IP          string   `json:"ip_address"`
}
