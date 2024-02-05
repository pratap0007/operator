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

import (
	"context"

	"knative.dev/pkg/ptr"

	"github.com/openshift-pipelines/pipelines-as-code/pkg/params/settings"
)

func (pac *OpenShiftPipelinesAsCode) SetDefaults(ctx context.Context) {
	if pac.Spec.PACSettings.Settings == nil {
		pac.Spec.PACSettings.Settings = map[string]string{}
	}
	if pac.Spec.PACSettings.AdditionalPACControllers == nil {
		pac.Spec.PACSettings.AdditionalPACControllers = map[string]AdditionalPACControllerConfig{}
	}
	pac.Spec.PACSettings.setPACDefaults()
}

func (set *PACSettings) setPACDefaults() {
	if set.Settings == nil {
		set.Settings = map[string]string{}
	}
	settings.SetDefaults(set.Settings)
	setAdditionalPACControllerDefault(set.AdditionalPACControllers)
}

// Set the default values for additional PAc controller resources
func setAdditionalPACControllerDefault(additionalPACController map[string]AdditionalPACControllerConfig) {
	for name, additionalPACInfo := range additionalPACController {
		if additionalPACInfo.Settings == nil {
			additionalPACInfo.Settings = map[string]string{}
		}
		settings.SetDefaults(additionalPACInfo.Settings)

		if additionalPACInfo.Enable == nil {
			additionalPACInfo.Enable = ptr.Bool(true)
		}
		if additionalPACInfo.ConfigMapName == "" {
			additionalPACInfo.ConfigMapName = name + "-configmap"
		}
		if additionalPACInfo.SecretName == "" {
			additionalPACInfo.SecretName = name + "-secret"
		}
		additionalPACController[name] = additionalPACInfo
	}
}
