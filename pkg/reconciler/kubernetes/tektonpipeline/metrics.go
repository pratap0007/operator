/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tektonpipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	// meterName is the name of the meter used for Tekton Pipeline metrics
	meterName = "tekton.dev/operator/pipeline"
)

var (
	// pReconcileCount is the counter for pipeline reconciliation
	pReconcileCount metric.Int64Counter

	// once ensures metrics are only initialized once
	once sync.Once

	// initErr stores any initialization error
	initErr error
)

// initMetrics initializes the OpenTelemetry metrics instruments
func initMetrics() error {
	once.Do(func() {
		meter := otel.Meter(meterName)

		pReconcileCount, initErr = meter.Int64Counter(
			"pipeline_reconcile_count",
			metric.WithDescription("number of pipeline install"),
			metric.WithUnit("1"),
		)
	})
	return initErr
}

// Recorder holds keys for Tekton metrics
type Recorder struct {
	initialized bool

	ReportingPeriod time.Duration
}

// NewRecorder creates a new metrics recorder instance
// to log the PipelineRun related metrics
func NewRecorder() (*Recorder, error) {
	r := &Recorder{
		initialized: true,

		// Default to 30s intervals.
		ReportingPeriod: 30 * time.Second,
	}

	if err := initMetrics(); err != nil {
		r.initialized = false
		return r, err
	}

	return r, nil
}

// Count logs number of times a component (pipeline/trigger atm)
// has been installed or failed to install.
func (r *Recorder) Count(status, version string) error {
	if !r.initialized {
		return fmt.Errorf(
			"ignoring the metrics recording for pipeline failed to initialize the metrics recorder")
	}

	// Record the metric with attributes
	pReconcileCount.Add(
		context.Background(),
		1,
		metric.WithAttributes(
			attribute.String("status", status),
			attribute.String("version", version),
		),
	)

	return nil
}

func (m *Recorder) LogMetrics(status, version string, logger *zap.SugaredLogger) {
	err := m.Count(status, version)
	if err != nil {
		logger.Warnf("%v: Failed to log the metrics : %v", resourceKind, err)
	}
}
