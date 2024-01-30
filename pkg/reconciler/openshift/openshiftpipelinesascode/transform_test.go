package openshiftpipelinesascode

import (
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/pipeline/test/diff"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestFilterAdditionalControllerManifest(t *testing.T) {
	testData := path.Join("testdata", "test-filter-manifest.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))
	assert.NilError(t, err)

	filteredManifest := filterAdditionalControllerManifest(manifest)
	assert.DeepEqual(t, len(filteredManifest.Resources()), 5)

	deployment := filteredManifest.Filter(mf.All(mf.ByKind("Deployment")))

	assert.DeepEqual(t, deployment.Resources()[0].GetName(), "pipelines-as-code-controller")

}

func TestUpdateAdditionControllerDeployment(t *testing.T) {
	testData := path.Join("testdata", "test-filter-manifest.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))
	assert.NilError(t, err)
	manifest = manifest.Filter(mf.All(mf.ByName("pipelines-as-code-controller"), mf.ByKind("Deployment")))

	additionalPACConfig := v1alpha1.AdditionalPACControllerConfig{
		ConfigMapName: "test-configmap",
		SecretName:    "test-secret",
	}
	updatedDeployment, err := manifest.Transform(updateAdditionControllerDeployment(&additionalPACConfig, "test"))
	assert.NilError(t, err)
	assert.DeepEqual(t, updatedDeployment.Resources()[0].GetName(), "test-controller")

	expectedData := path.Join("testdata", "test-expected-additional-pac-dep.yaml")
	expectedManifest, err := mf.ManifestFrom(mf.Recursive(expectedData))
	assert.NilError(t, err)

	expected := &appsv1.Deployment{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(expectedManifest.Resources()[0].Object, expected)
	if err != nil {
		assert.NilError(t, err)
	}

	got := &appsv1.Deployment{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedDeployment.Resources()[0].Object, got)
	if err != nil {
		assert.NilError(t, err)
	}

	if d := cmp.Diff(got, expected); d != "" {
		t.Errorf("failed to update additional pac controller deployment %s", diff.PrintWantGot(d))
	}

}

func TestUpdateAdditionControllerService(t *testing.T) {
	testData := path.Join("testdata", "test-filter-manifest.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))
	assert.NilError(t, err)
	manifest = manifest.Filter(mf.All(mf.ByName("pipelines-as-code-controller"), mf.ByKind("Service")))

	additionalPACConfig := v1alpha1.AdditionalPACControllerConfig{
		ConfigMapName: "test-configmap",
		SecretName:    "test-secret",
	}
	updatedManifest, err := manifest.Transform(updateAdditionControllerService(&additionalPACConfig, "test"))
	assert.NilError(t, err)
	assert.DeepEqual(t, updatedManifest.Resources()[0].GetName(), "test-controller")
}

func TestUpdateAdditionControllerRoute(t *testing.T) {
	testData := path.Join("testdata", "test-filter-manifest.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))
	assert.NilError(t, err)
	manifest = manifest.Filter(mf.All(mf.ByName("pipelines-as-code-controller"), mf.ByKind("Route")))

	additionalPACConfig := v1alpha1.AdditionalPACControllerConfig{
		ConfigMapName: "test-configmap",
		SecretName:    "test-secret",
	}
	updatedManifest, err := manifest.Transform(updateAdditionControllerRoute(&additionalPACConfig, "test"))
	assert.NilError(t, err)
	assert.DeepEqual(t, updatedManifest.Resources()[0].GetName(), "test-controller")
}

func TestUpdateAdditionControllerServiceMonitor(t *testing.T) {
	testData := path.Join("testdata", "test-filter-manifest.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))
	assert.NilError(t, err)
	manifest = manifest.Filter(mf.All(mf.ByName("pipelines-as-code-controller-monitor"), mf.ByKind("ServiceMonitor")))

	additionalPACConfig := v1alpha1.AdditionalPACControllerConfig{
		ConfigMapName: "test-configmap",
		SecretName:    "test-secret",
	}
	updatedManifest, err := manifest.Transform(updateAdditionControllerServiceMonitor(&additionalPACConfig, "test"))
	assert.NilError(t, err)
	assert.DeepEqual(t, updatedManifest.Resources()[0].GetName(), "test-controller")
}

func TestUpdateAdditionControllerConfigMapWithDefaultCM(t *testing.T) {
	testData := path.Join("testdata", "test-filter-manifest.yaml")
	manifest, err := mf.ManifestFrom(mf.Recursive(testData))
	assert.NilError(t, err)
	manifest = manifest.Filter(mf.All(mf.ByName("pipelines-as-code"), mf.ByKind("ConfigMap")))

	additionalPACConfig := v1alpha1.AdditionalPACControllerConfig{
		ConfigMapName: "",
		SecretName:    "test-secret",
	}
	updatedManifest, err := manifest.Transform(updateAdditionControllerConfigMap(&additionalPACConfig, "test"))
	assert.NilError(t, err)
	assert.DeepEqual(t, updatedManifest.Resources()[0].GetName(), "pipelines-as-code")
}
