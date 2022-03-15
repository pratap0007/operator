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

package tektonconfig

import (
	"context"

	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createInstallerSet(ctx context.Context, oc versioned.Interface, tc *v1alpha1.TektonConfig, labels map[string]string,
	releaseVersion, component, installerSetName string) error {

	is := makeInstallerSet(tc, installerSetName, releaseVersion, labels)

	createdIs, err := oc.OperatorV1alpha1().TektonInstallerSets().
		Create(ctx, is, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	if len(tc.Status.TektonInstallerSet) == 0 {
		tc.Status.TektonInstallerSet = map[string]string{}
	}

	// Update the status of tektonConfig with created installerSet name
	tc.Status.TektonInstallerSet[component] = createdIs.Name
	tc.Status.SetVersion(releaseVersion)
	return nil
}

func makeInstallerSet(tc *v1alpha1.TektonConfig, name, releaseVersion string, labels map[string]string) *v1alpha1.TektonInstallerSet {
	ownerRef := *metav1.NewControllerRef(tc, tc.GetGroupVersionKind())
	return &v1alpha1.TektonInstallerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
			Annotations: map[string]string{
				v1alpha1.ReleaseVersionKey:  releaseVersion,
				v1alpha1.TargetNamespaceKey: tc.Spec.TargetNamespace,
			},
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
	}
}

func deleteInstallerSet(ctx context.Context, oc versioned.Interface, tc *v1alpha1.TektonConfig, component string) error {

	compInstallerSet, ok := tc.Status.TektonInstallerSet[component]
	if !ok || compInstallerSet == "" {
		return nil
	}

	// delete the installer set
	err := oc.OperatorV1alpha1().TektonInstallerSets().
		Delete(ctx, tc.Status.TektonInstallerSet[component], metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	// clear the name of installer set from TektonConfig status
	delete(tc.Status.TektonInstallerSet, component)

	return nil
}

// checkIfInstallerSetExist checks if installer set exists for a component and return true/false based on it
// and if installer set which already exist is of older version then it deletes and return false to create a new
// installer set
func checkIfInstallerSetExist(ctx context.Context, oc versioned.Interface, relVersion string,
	tc *v1alpha1.TektonConfig, component string) (*v1alpha1.TektonInstallerSet, error) {

	// Check if installer set is already created
	compInstallerSet, ok := tc.Status.TektonInstallerSet[component]
	if !ok || compInstallerSet == "" {
		return nil, nil
	}

	// if already created then check which version it is
	ctIs, err := oc.OperatorV1alpha1().TektonInstallerSets().
		Get(ctx, compInstallerSet, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	if version, ok := ctIs.Annotations[v1alpha1.ReleaseVersionKey]; ok && version == relVersion {
		// if installer set already exist and release version is same
		// then ignore and move on
		return ctIs, nil
	}

	// release version doesn't exist or is different from expected
	// deleted existing InstallerSet and create a new one

	err = oc.OperatorV1alpha1().TektonInstallerSets().
		Delete(ctx, compInstallerSet, metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return nil, v1alpha1.RECONCILE_AGAIN_ERR
}
