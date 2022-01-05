package collector

import (
	"encoding/json"
	"fmt"
	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/songjiayang/nginx-log-exporter/config"
	"log"
	"strconv"
	"strings"
)

// Collector is a struct containing pointers to all metrics that should be
// exposed to Prometheus
type Collector struct {
	countTotal      *prometheus.CounterVec
	bytesTotal      *prometheus.CounterVec
	upstreamSeconds *prometheus.HistogramVec
	responseSeconds *prometheus.HistogramVec

	externalValues  []string
	dynamicLabels   []string
	dynamicValueLen int

	cfg    *config.AppConfig
}

func NewCollector(cfg *config.AppConfig) *Collector {
	exlables, exValues := cfg.ExternalLabelSets()
	dynamicLabels := cfg.DynamicLabels()

	labels := append(exlables, dynamicLabels...)

	return &Collector{
		countTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Name,
			Name:      "http_response_count_total",
			Help:      "Amount of processed HTTP requests",
		}, labels),

		bytesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: cfg.Name,
			Name:      "http_response_size_bytes",
			Help:      "Total amount of transferred bytes",
		}, labels),

		upstreamSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Name,
			Name:      "http_upstream_time_seconds",
			Help:      "Time needed by upstream servers to handle requests",
			Buckets:   cfg.HistogramBuckets,
		}, labels),

		responseSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: cfg.Name,
			Name:      "http_response_time_seconds",
			Help:      "Time needed by NGINX to handle requests",
			Buckets:   cfg.HistogramBuckets,
		}, labels),

		externalValues:  exValues,
		dynamicLabels:   dynamicLabels,
		dynamicValueLen: len(dynamicLabels),

		cfg:    cfg,
	}
}

func (c *Collector) Run() {
	c.cfg.Prepare()

	// register to prometheus
	prometheus.MustRegister(c.countTotal)
	prometheus.MustRegister(c.bytesTotal)
	prometheus.MustRegister(c.upstreamSeconds)
	prometheus.MustRegister(c.responseSeconds)

	for _, f := range c.cfg.SourceFiles {
		t, err := tail.TailFile(f, tail.Config{
			Follow: true,
			ReOpen: true,
			Poll:   true,
		})

		if err != nil {
			log.Panic(err)
		}

		go func() {
			var data config.JsonFormat
			for line := range t.Lines {
				err := json.Unmarshal([]byte(line.Text), &data)
				if err != nil {
					fmt.Printf("error while parsing line '%s': %s", line.Text, err)
					continue
				}

				dynamicValues := make([]string, c.dynamicValueLen)

				dynamicValues = []string{
					c.formatValue("componentName", data.ComponentName),
					c.formatValue("interfaceName", data.InterfaceName),
					c.formatValue("returnCode", strconv.Itoa(data.ReturnCode)),
				}

				labelValues := append(c.externalValues, dynamicValues...)

				c.countTotal.WithLabelValues(labelValues...).Inc()

				c.updateHistogramMetric(c.upstreamSeconds, labelValues, "upstream_response_time", float64(data.Timestamp))
				c.updateHistogramMetric(c.responseSeconds, labelValues, "request_time", float64(data.Timestamp + data.CostTime))
			}
		}()
	}
}

func (c *Collector) formatValue(label, value string) string {
	replacement, ok := c.cfg.RelabelConfig.Replacement[label]
	if !ok {
		return value
	}

	if replacement.Trim != "" {
		value = strings.Split(value, replacement.Trim)[0]
	}

	for _, target := range replacement.Replaces {
		if target.Regexp().MatchString(value) {
			return target.Value
		}
	}

	return value
}

func (c *Collector) updateHistogramMetric(metric *prometheus.HistogramVec, labelValues []string, name string, timestamp float64) {

	exemplarLabels := prometheus.Labels{}
	exemplarLabels[name] = name

	metric.WithLabelValues(labelValues...).(prometheus.ExemplarObserver).ObserveWithExemplar(
		timestamp, exemplarLabels,
	)
}
