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

package tektonchain

import (
	"testing"

	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
)

func TestUninitializedMetrics(t *testing.T) {
	recorder := Recorder{initialized: false}
	spec := v1alpha1.TektonChainSpec{}
	if err := recorder.Count("v0.1", spec); err == nil {
		t.Error("recorder.Count expected to return error for uninitialized recorder, but got nil")
	}
}

func TestMetricsCount(t *testing.T) {
	recorder, err := NewRecorder()
	if err != nil {
		t.Errorf("failed to initialize recorder, got %s", err.Error())
	}

	// Test with empty spec
	spec := v1alpha1.TektonChainSpec{}
	if err := recorder.Count("v0.20.0", spec); err != nil {
		t.Errorf("recorder.Count recording failed got %s", err.Error())
	}

	// Test with populated spec using the correct embedded structure
	taskrunStorage := "oci"
	pipelinerunStorage := "oci"
	ociStorage := "oci"
	spec = v1alpha1.TektonChainSpec{
		Chain: v1alpha1.Chain{
			ChainProperties: v1alpha1.ChainProperties{
				ArtifactsTaskRunFormat:      "in-toto",
				ArtifactsTaskRunStorage:     &taskrunStorage,
				ArtifactsTaskRunSigner:      "x509",
				ArtifactsPipelineRunFormat:  "in-toto",
				ArtifactsPipelineRunStorage: &pipelinerunStorage,
				ArtifactsPipelineRunSigner:  "x509",
				ArtifactsOCIFormat:          "simplesigning",
				ArtifactsOCIStorage:         &ociStorage,
				ArtifactsOCISigner:          "x509",
			},
		},
	}
	if err := recorder.Count("v0.20.0", spec); err != nil {
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
