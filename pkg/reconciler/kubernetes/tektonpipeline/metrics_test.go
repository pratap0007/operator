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
	"testing"
)

func TestUninitializedMetrics(t *testing.T) {
	recorder := Recorder{initialized: false}
	if err := recorder.Count("installed", "v0.1"); err == nil {
		t.Error("recorder.Count expected to return error for uninitialized recorder, but got nil")
	}
}

func TestMetricsCount(t *testing.T) {
	recorder, err := NewRecorder()
	if err != nil {
		t.Errorf("failed to initialize recorder, got %s", err.Error())
	}

	// Test that Count doesn't return an error for initialized recorder
	if err := recorder.Count("installed", "v0.50.0"); err != nil {
		t.Errorf("recorder.Count recording failed got %s", err.Error())
	}

	// Test with different status
	if err := recorder.Count("failed", "v0.50.0"); err != nil {
		t.Errorf("recorder.Count recording failed got %s", err.Error())
	}
}

func TestNewRecorder(t *testing.T) {
	recorder, err := NewRecorder()
	if err != nil {
		t.Errorf("NewRecorder() returned error: %v", err)
	}

	if recorder == nil {
		t.Error("NewRecorder() returned nil recorder")
	}

	if !recorder.initialized {
		t.Error("NewRecorder() returned uninitialized recorder")
	}
}
