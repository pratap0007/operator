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
	"fmt"

	"github.com/openshift-pipelines/pipelines-as-code/pkg/params/settings"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

// this is 25 because this name goes in the installerset name which already have 38 characters, so additional length we
// can have for name is 25, as the kibernetes have restriction for 63
const additionalPACControllerNameLength = 25

func (pac *OpenShiftPipelinesAsCode) Validate(ctx context.Context) *apis.FieldError {
	if apis.IsInDelete(ctx) {
		return nil
	}

	var errs *apis.FieldError

	// execute common spec validations
	errs = errs.Also(pac.Spec.CommonSpec.validate("spec"))

	errs = errs.Also(validatePACSetting(pac.Spec.PACSettings))

	return errs
}

func validatePACSetting(pacSettings PACSettings) *apis.FieldError {
	var errs *apis.FieldError

	if err := settings.Validate(pacSettings.Settings); err != nil {
		errs = errs.Also(apis.ErrInvalidValue(err, "spec.platforms.openshift.pipelinesAsCode.PACSettings"))
	}

	errs = errs.Also(validateAdditionalPACSetting(pacSettings.AdditionalPACControllers))

	return errs
}

func validateAdditionalPACSetting(additionalPACController map[string]AdditionalPACControllerConfig) *apis.FieldError {
	var errs *apis.FieldError
	for name, additionalPACInfo := range additionalPACController {

		if err := validateControllerName(name); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(err, "spec.platforms.openshift.pipelinesAsCode.PACSettings.AdditionalPACControllers"))
		}

		if err := validateName(additionalPACInfo.ConfigMapName); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(err, "spec.platforms.openshift.pipelinesAsCode.PACSettings.AdditionalPACControllers.ConfigMapName"))
		}

		if err := validateName(additionalPACInfo.SecretName); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(err, "spec.platforms.openshift.pipelinesAsCode.PACSettings.AdditionalPACControllers.SecretName"))
		}

		if err := settings.Validate(additionalPACInfo.Settings); err != nil {
			errs = errs.Also(apis.ErrInvalidValue(err, "spec.platforms.openshift.pipelinesAsCode.PACSettings.AdditionalPACControllers.Settings"))
		}
	}

	return errs
}

// validates the name of the controller resource is valid kubernetes name
func validateControllerName(name string) *apis.FieldError {
	if err := validation.IsDNS1123Subdomain(name); len(err) > 0 {
		return &apis.FieldError{
			Message: fmt.Sprintf("invalid resource name %q: must be a valid DNS label", name),
			Paths:   []string{"name"},
		}
	}

	if len(name) > additionalPACControllerNameLength {
		return &apis.FieldError{
			Message: fmt.Sprintf("invalid resource name %q: length must be no more than %d characters", name, additionalPACControllerNameLength),
			Paths:   []string{"name"},
		}
	}
	return nil
}

// validates the name of the resource is valid kubernetes name
func validateName(name string) *apis.FieldError {
	if err := validation.IsDNS1123Subdomain(name); len(err) > 0 {
		return &apis.FieldError{
			Message: fmt.Sprintf("invalid resource name %q: must be a valid DNS label", name),
			Paths:   []string{"name"},
		}
	}

	if len(name) > validation.DNS1123LabelMaxLength {
		return &apis.FieldError{
			Message: fmt.Sprintf("invalid resource name %q: length must be no more than %d characters", name, validation.DNS1123LabelMaxLength),
			Paths:   []string{"name"},
		}
	}
	return nil
}
