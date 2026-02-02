/*
Copyright 2023 The Tekton Authors

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

package tektonresult

import (
	"context"
	"fmt"
	"sync"

	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	// meterName is the name of the meter used for Tekton Result metrics
	meterName = "tekton.dev/operator/result"
)

var (
	// rReconcileCount is the gauge for results reconciliation
	rReconcileCount metric.Int64Gauge

	// once ensures metrics are only initialized once
	once sync.Once

	// initErr stores any initialization error
	initErr error

	errUninitializedRecorder = fmt.Errorf("ignoring the metrics recording for result failed to initialize the metrics recorder")
)

// initMetrics initializes the OpenTelemetry metrics instruments
func initMetrics() error {
	once.Do(func() {
		meter := otel.Meter(meterName)

		// Using Int64Gauge to match the original LastValue aggregation behavior
		rReconcileCount, initErr = meter.Int64Gauge(
			"results_reconciled",
			metric.WithDescription("results reconciled with their log type"),
			metric.WithUnit("1"),
		)
	})
	return initErr
}

// Recorder holds keys for Tekton metrics
type Recorder struct {
	initialized bool
}

// NewRecorder creates a new metrics recorder instance
// to log the PipelineRun related metrics
func NewRecorder() (*Recorder, error) {
	r := &Recorder{
		initialized: true,
	}

	if err := initMetrics(); err != nil {
		r.initialized = false
		return r, err
	}

	return r, nil
}

// Record the Results reconciled with their log type
func (r *Recorder) Count(version, logType string) error {
	if !r.initialized {
		return errUninitializedRecorder
	}

	// Record the metric with attributes
	rReconcileCount.Record(
		context.Background(),
		1,
		metric.WithAttributes(
			attribute.String("version", version),
			attribute.String("log_type", logType),
		),
	)

	return nil
}

func (m *Recorder) LogMetrics(version string, spec v1alpha1.TektonResultSpec, logger *zap.SugaredLogger) {
	err := m.Count(version, spec.Result.ResultsAPIProperties.LogsType)
	if err != nil {
		logger.Warnf("%v: Failed to log the metrics : %v", v1alpha1.KindTektonResult, err)
	}
}

// RecorderWrapper wraps the existing Recorder to implement this interface.
type RecorderWrapper struct {
	recorder *Recorder
}

// NewRecorderWrapper creates a new RecorderWrapper instance.
func NewRecorderWrapper(recorder *Recorder) *RecorderWrapper {
	return &RecorderWrapper{recorder: recorder}
}

// LogMetrics implements the Metrics interface by converting the provided logType string
// into a TektonResultSpec before calling the underlying Recorder's LogMetrics method.
func (rw *RecorderWrapper) LogMetrics(logType string, version string, logger *zap.SugaredLogger) {
	spec := v1alpha1.TektonResultSpec{
		Result: v1alpha1.Result{
			ResultsAPIProperties: v1alpha1.ResultsAPIProperties{
				LogsType: logType,
			},
		},
	}
	rw.recorder.LogMetrics(version, spec, logger)
}
