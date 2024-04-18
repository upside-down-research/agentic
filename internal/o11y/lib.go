package o11y

import (
	"context"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

var pusher *push.Pusher

type MetricManager struct {
	labelNames []string
	gauges     *prometheus.GaugeVec
	metrics    map[string]prometheus.Gauge
	mu         sync.Mutex
}

func NewMetricManager(name, help string, labelNames []string) *MetricManager {
	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labelNames,
	)
	return &MetricManager{
		gauges:     g,
		labelNames: labelNames,
		metrics:    make(map[string]prometheus.Gauge),
	}
}

var mm *MetricManager
var LlmCounter *prometheus.CounterVec

func init() {
	// Set gauge to some value
	pusher = push.New("http://localhost:9091", "agentic_pusher") // replace "localhost:9091" with your Pushgateway address
	mm = NewMetricManager("llm_duration", "LLM duration", []string{"model"})
	LlmCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_counter",
		},
		[]string{"model", "id", "job_name"})
	pusher.Collector(LlmCounter)
}

func isUnorderedEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Strings(a)
	sort.Strings(b)
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (m *MetricManager) GetGauge(labelValues map[string]string) prometheus.Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	// read keys of labelValues
	var keys []string
	for k := range labelValues {
		keys = append(keys, k)
	}
	// compare keys to labelNames
	if !isUnorderedEqual(keys, m.labelNames) {
		log.Fatal("labelNames do not match labelValues")
	}

	// Create a key by concatenating all label values
	key := m.createKey(labelValues)

	// Check if the gauge already exists
	if gauge, exists := m.metrics[key]; exists {
		return gauge
	}

	// Create a new gauge with the specified labels
	gauge := m.gauges.With(labelValues)
	m.metrics[key] = gauge
	// register the gauge with the pusher
	pusher.Collector(gauge)
	return gauge
}

func (m *MetricManager) createKey(labelValues map[string]string) string {
	var labels []string
	for _, v := range labelValues {
		labels = append(labels, v)
	}
	sort.Strings(labels)
	return strings.Join(labels, "|")
}

func WriteData(name string, tags map[string]string, data float32) {

	mm.GetGauge(tags).Set(float64(data))
	// launch a goroutine to do the pushing
	go func() {
		err := pusher.Push()
		if err != nil {
			log.Println("Error pushing data to Pushgateway:", err)
			return
		}
	}()
}
func Record(name string, tags map[string]string, fields map[string]interface{}) {
	token := "iWtWImS2XKfzNSmeNg-jNXZh-2Zmjksr5iMDGQBu5teUkPBu5IEClt4FGRcLRQDFVhW_Aug5XCBM8PHD8KnmSw=="
	url := "http://localhost:8086"
	client := influxdb2.NewClient(url, token)
	org := "udr"
	bucket := "bucket1"
	writeAPI := client.WriteAPIBlocking(org, bucket)
	point := write.NewPoint(name, tags, fields, time.Now())
	if err := writeAPI.WritePoint(context.Background(), point); err != nil {
		log.Fatal(err)
	}
}
