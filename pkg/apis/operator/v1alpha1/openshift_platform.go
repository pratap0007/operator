/*
Copyright 2022 The Tekton Authors

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

package v1alpha1

type OpenShift struct {
	// PipelinesAsCode allows configuring PipelinesAsCode configurations
	// +optional
	PipelinesAsCode *PipelinesAsCode `json:"pipelinesAsCode,omitempty"`
	// SCC allows configuring security context constraints used by workloads
	// +optional
	SCC *SCC `json:"scc,omitempty"`
}

type PipelinesAsCode struct {
	// Enable or disable pipelines as code by changing this bool
	// +optional
	Enable *bool `json:"enable,omitempty"`
	// PACSettings allows user to configure PAC configurations
	// +optional
	PACSettings `json:",inline"`
	// AdditionalPACControllers allows to add multiple PipelineAsCode configuration
	// +optional
	AdditionalPACControllers map[string]AdditionalPACControllerConfig `json:"additionalPACControllers"`
}

type SCC struct {
	// Default contains the default SCC that will be attached to the service
	// account used for workloads (`pipeline` SA by default) and defined in
	// PipelineProperties.OptionalPipelineProperties.DefaultServiceAccount
	// +optional
	Default string `json:"default,omitempty"`
	// MaxAllowed specifies the highest SCC that can be requested for in a
	// namespace or in the Default field.
	// +optional
	MaxAllowed string `json:"maxAllowed,omitempty"`
}

// This contains AdditionalPACControllerConfig
type AdditionalPACControllerConfig struct {
	Name          string `json:"name"`                    //name of the controller
	ConfigMapName string `json:"configMapName,omitempty"` // configMap name
	SecretsName   string `json:"secretsName,omitempty"`   // Secrets name
}
