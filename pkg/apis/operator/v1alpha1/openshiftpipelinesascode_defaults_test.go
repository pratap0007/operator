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
	"testing"

	"knative.dev/pkg/ptr"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetAdditionalPACControllerDefault(t *testing.T) {
	opacCR := &OpenShiftPipelinesAsCode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: OpenShiftPipelinesAsCodeSpec{
			PACSettings: PACSettings{
				Settings: map[string]string{},
				AdditionalPACControllers: map[string]AdditionalPACControllerConfig{
					"test": {},
				},
			},
		},
	}

	opacCR.Spec.PACSettings.setPACDefaults()

	assert.Equal(t, *opacCR.Spec.PACSettings.AdditionalPACControllers["test"].Enable, true)
	assert.Equal(t, opacCR.Spec.PACSettings.AdditionalPACControllers["test"].ConfigMapName, "test-configmap")
	assert.Equal(t, opacCR.Spec.PACSettings.AdditionalPACControllers["test"].SecretName, "test-secret")
}

func TestSetAdditionalPACControllerDefaultHavingAdditionalPACController(t *testing.T) {
	opacCR := &OpenShiftPipelinesAsCode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "name",
			Namespace: "namespace",
		},
		Spec: OpenShiftPipelinesAsCodeSpec{
			PACSettings: PACSettings{
				Settings: map[string]string{},
				AdditionalPACControllers: map[string]AdditionalPACControllerConfig{
					"test": {
						Enable:        ptr.Bool(false),
						ConfigMapName: "test-configmap",
						SecretName:    "test-secret",
						Settings: map[string]string{
							"application-name":    "Additional PACController CI",
							"custom-console-name": "custom",
							"custom-console-url":  "https://custom.com",
						},
					},
				},
			},
		},
	}

	opacCR.Spec.PACSettings.setPACDefaults()

	assert.Equal(t, *opacCR.Spec.PACSettings.AdditionalPACControllers["test"].Enable, false)
	assert.Equal(t, opacCR.Spec.PACSettings.AdditionalPACControllers["test"].Settings["application-name"], "Additional PACController CI")
	assert.Equal(t, opacCR.Spec.PACSettings.AdditionalPACControllers["test"].Settings["custom-console-name"], "custom")
	assert.Equal(t, opacCR.Spec.PACSettings.AdditionalPACControllers["test"].Settings["custom-console-url"], "https://custom.com")
}
