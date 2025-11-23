package packet

import "time"

type Metric struct {
	Name   string
	Labels map[string]string
	Value  float64
	Time   time.Time
}

type Metricer interface {
	ToMetric() Metric
}
