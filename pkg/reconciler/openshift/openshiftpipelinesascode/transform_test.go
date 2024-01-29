package openshiftpipelinesascode

import (
	"path"
	"testing"

	"github.com/google/go-cmp/cmp"
	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/pipeline/test/diff"
	"gotest.tools/v3/assert"
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

	expectedData := path.Join("testdata", "test-expected-additional-pac-dep.yaml")
	expectedManifest, err := mf.ManifestFrom(mf.Recursive(expectedData))
	assert.NilError(t, err)

	deployment := manifest.Filter(mf.All(mf.ByName("pipelines-as-code-controller"), mf.ByKind("Deployment")))

	additionalPACConfig := v1alpha1.AdditionalPACControllerConfig{
		ConfigMapName: "test-configmap",
		SecretName:    "test-secret",
	}

	updatedDeployment, err := deployment.Transform(updateAdditionControllerDeployment(&additionalPACConfig, "test"))
	assert.NilError(t, err)
	assert.DeepEqual(t, updatedDeployment.Resources()[0].GetName(), "test-controller")

	if d := cmp.Diff(expectedManifest.Resources(), updatedDeployment.Resources()); d != "" {
		t.Errorf("failed to update additional pac controller deployment %s", diff.PrintWantGot(d))
	}

}
