/*
Copyright 2024 The Tekton Authors

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

package resources

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	mfc "github.com/manifestival/client-go-client"
	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	typedv1alpha1 "github.com/tektoncd/operator/pkg/client/clientset/versioned/typed/operator/v1alpha1"
	"github.com/tektoncd/operator/test/utils"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/test/logging"
)

// EnsureOpenShiftPipelinesAsCodeExists creates a OpenShiftPipelinesAsCode with the name names.OpenShiftPipelinesAsCode, if it does not exist.
func EnsureOpenShiftPipelinesAsCodeExists(clients typedv1alpha1.OpenShiftPipelinesAsCodeInterface, names utils.ResourceNames) (*v1alpha1.OpenShiftPipelinesAsCode, error) {
	// If this function is called by the upgrade tests, we only create the custom resource, if it does not exist.
	ks, err := clients.Get(context.TODO(), names.OpenShiftPipelinesAsCode, metav1.GetOptions{})
	if apierrs.IsNotFound(err) {
		ks := &v1alpha1.OpenShiftPipelinesAsCode{
			ObjectMeta: metav1.ObjectMeta{
				Name: names.OpenShiftPipelinesAsCode,
			},
			Spec: v1alpha1.OpenShiftPipelinesAsCodeSpec{
				CommonSpec: v1alpha1.CommonSpec{
					TargetNamespace: names.TargetNamespace,
				},
				PACSettings: v1alpha1.PACSettings{
					Settings: map[string]string{},
				},
			},
		}
		return clients.Create(context.TODO(), ks, metav1.CreateOptions{})
	}
	return ks, err
}

// Fetch the OpenShiftPipelinesAsCode CR and update the
func CreateAdditionalPipelinesASCodeController(clients typedv1alpha1.OpenShiftPipelinesAsCodeInterface, names utils.ResourceNames) (*v1alpha1.OpenShiftPipelinesAsCode, error) {
	// If this function is called by the upgrade tests, we only create the custom resource, if it does not exist.
	opacCR, _ := clients.Get(context.TODO(), names.OpenShiftPipelinesAsCode, metav1.GetOptions{})
	enable := true

	// update the OpenshiftPipelines CR to add the additional Pipelines As Code Controller
	opacCR.Spec.AdditionalPACControllers = map[string]v1alpha1.AdditionalPACControllerConfig{
		"test": {
			Enable:        &enable,
			ConfigMapName: "test-configmap",
			SecretName:    "test-secret",
			Settings: map[string]string{
				"application-name":                    "Pipelines as Code CI",
				"auto-configure-new-github-repo":      "false",
				"bitbucket-cloud-check-source-ip":     "true",
				"custom-console-name":                 "",
				"custom-console-url":                  "",
				"custom-console-url-pr-details":       "",
				"custom-console-url-pr-tasklog":       "",
				"error-detection-from-container-logs": "false",
				"error-detection-max-number-of-lines": "50",
				"error-detection-simple-regexp": `^(?P<filename>[^:]*):(?P<line>[0-9]+):(?P<column>[0-9]+):([
				  ]*)?(?P<error>.*)`,
				"error-log-snippet":              "true",
				"hub-catalog-name":               "tekton",
				"hub-url":                        "https://api.hub.tekton.dev/v1",
				"remote-tasks":                   "true",
				"secret-auto-create":             "true",
				"secret-github-app-token-scoped": "true",
			},
		},
	}
	return clients.Update(context.TODO(), opacCR, metav1.UpdateOptions{})
}

// WaitForOpenshiftPipelinesAsCodeState polls the status of the Pipelines As Code called name
// from client every `interval` until `inState` returns `true` indicating it
// is done, returns an error or timeout.
func WaitForOpenshiftPipelinesAsCode(clients typedv1alpha1.OpenShiftPipelinesAsCodeInterface, name string,
	inState func(s *v1alpha1.OpenShiftPipelinesAsCode, err error) (bool, error)) (*v1alpha1.OpenShiftPipelinesAsCode, error) {
	span := logging.GetEmitableSpan(context.Background(), fmt.Sprintf("WaitForOpenShiftPipelinesAsCodeState/%s/%s", name, "TektonOpenShiftPipelinesASCodeIsReady"))
	defer span.End()

	var lastState *v1alpha1.OpenShiftPipelinesAsCode
	waitErr := wait.PollImmediate(utils.Interval, utils.Timeout, func() (bool, error) {
		lastState, err := clients.Get(context.TODO(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("OpenShiftPipelinesAsCode %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

// IsOpenShiftPipelinesAsCodeReady will check the status conditions of the OpenShiftPipelinesAsCode and return true if the OpenShiftPipelinesASCode is ready.
func IsOpenShiftPipelinesAsCodeReady(s *v1alpha1.OpenShiftPipelinesAsCode, err error) (bool, error) {
	return s.Status.IsReady(), err
}

// AssertOpenShiftPIpelinesCRReadyStatus verifies if the OpenShiftPIpelinesAsCode reaches the READY status.
func AssertOpenShiftPipelinesAsCodeCRReadyStatus(t *testing.T, clients *utils.Clients, names utils.ResourceNames) {
	if _, err := WaitForOpenshiftPipelinesAsCode(clients.OpenShiftPipelinesAsCode(), names.OpenShiftPipelinesAsCode, IsOpenShiftPipelinesAsCodeReady); err != nil {
		t.Fatalf("OpenShiftPIpelinesAsCodeCR %q failed to get to the READY status: %v", names.OpenShiftPipelinesAsCode, err)
	}
}

// OpenShiftPipelinesASCodeCRDelete deletes tha OpenShiftPipelinesAsCode to see if all resources will be deleted
func OpenShiftPipelinesAsCodeCRDelete(t *testing.T, clients *utils.Clients, crNames utils.ResourceNames) {
	if err := clients.OpenShiftPipelinesAsCode().Delete(context.TODO(), crNames.OpenShiftPipelinesAsCode, metav1.DeleteOptions{}); err != nil {
		t.Fatalf("OpenShiftPipelinesAsCode %q failed to delete: %v", crNames.OpenShiftPipelinesAsCode, err)
	}
	err := wait.PollImmediate(utils.Interval, utils.Timeout, func() (bool, error) {
		_, err := clients.OpenShiftPipelinesAsCode().Get(context.TODO(), crNames.OpenShiftPipelinesAsCode, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		t.Fatal("Timed out waiting on OpenShiftPipelinesCode to delete", err)
	}
	_, b, _, _ := runtime.Caller(0)
	m, err := mfc.NewManifest(filepath.Join((filepath.Dir(b)+"/.."), "manifests/"), clients.Config)
	if err != nil {
		t.Fatal("Failed to load manifest", err)
	}
	if err := verifyNoOpenShiftPipelinesAsCodeCR(clients); err != nil {
		t.Fatal(err)
	}

	// verify all but the CRD's and the Namespace are gone
	for _, u := range m.Filter(mf.NoCRDs, mf.Not(mf.ByKind("Namespace"))).Resources() {
		if _, err := m.Client.Get(&u); !apierrs.IsNotFound(err) {
			t.Fatalf("The %s %s failed to be deleted: %v", u.GetKind(), u.GetName(), err)
		}
	}
	// verify all the CRD's remain
	for _, u := range m.Filter(mf.CRDs).Resources() {
		if _, err := m.Client.Get(&u); apierrs.IsNotFound(err) {
			t.Fatalf("The %s CRD was deleted", u.GetName())
		}
	}
}

func IsAdditionalPACDeploymentAvailable(kubeClient kubernetes.Interface, name, namespace string) (bool, error) {
	dep, err := kubeClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
		return false, err
	}

	return IsDeploymentAvailable(dep)
}

// AssertOpenShiftPIpelinesCRReadyStatus verifies if the OpenShiftPIpelinesAsCode reaches the READY status.
func AssertAdditionalPACControllerDeletion(t *testing.T, kubeClient kubernetes.Interface, name string, namespace string) {
	_, err := kubeClient.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err == nil {
		t.Fatalf("failed to delete additional PAC controller deployment %q: %v", name, err)
	}
}

// Fetch the OpenShiftPipelinesAsCode CR and and delete the additional pipelines as code config
func RemoveAdditionalPipelinesASCodeController(clients typedv1alpha1.OpenShiftPipelinesAsCodeInterface, names utils.ResourceNames) (*v1alpha1.OpenShiftPipelinesAsCode, error) {
	// If this function is called by the upgrade tests, we only create the custom resource, if it does not exist.
	opacCR, _ := clients.Get(context.TODO(), names.OpenShiftPipelinesAsCode, metav1.GetOptions{})

	// update the OpenshiftPipelines CR to add the additional Pipelines As Code Controller
	opacCR.Spec.AdditionalPACControllers = map[string]v1alpha1.AdditionalPACControllerConfig{}
	return clients.Update(context.TODO(), opacCR, metav1.UpdateOptions{})
}
func verifyNoOpenShiftPipelinesAsCodeCR(clients *utils.Clients) error {
	opacCR, err := clients.OpenShiftPipelinesAsCode().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(opacCR.Items) > 0 {
		return errors.New("unable to verify cluster-scoped resources are deleted if any OpenShiftPipelinesAsCode exists")
	}
	return nil
}
