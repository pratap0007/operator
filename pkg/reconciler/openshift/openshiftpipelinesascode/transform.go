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

package openshiftpipelinesascode

import (
	"context"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"github.com/tektoncd/operator/pkg/reconciler/kubernetes/tektoninstallerset/client"
	"github.com/tektoncd/operator/pkg/reconciler/openshift"
	occommon "github.com/tektoncd/operator/pkg/reconciler/openshift/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const pipelinesAsCodeCM = "pipelines-as-code"
const additionalPACControllerSuffix = "-controller"

// const additionPACControllerConfigMap = "addition-pac"

func filterAndTransform(extension common.Extension) client.FilterAndTransform {
	return func(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.TektonComponent) (*mf.Manifest, error) {
		pac := comp.(*v1alpha1.OpenShiftPipelinesAsCode)
		// installerSet adds it's owner as namespace's owner
		// so deleting tekton addon deletes target namespace too
		// to skip it we filter out namespace
		pacManifest := manifest.Filter(mf.Not(mf.ByKind("Namespace")))

		images := common.ToLowerCaseKeys(common.ImagesFromEnv(common.PacImagePrefix))
		// Run transformers
		tfs := []mf.Transformer{
			common.InjectOperandNameLabelOverwriteExisting(openshift.OperandOpenShiftPipelineAsCode),
			common.DeploymentImages(images),
			common.AddConfiguration(pac.Spec.Config),
			occommon.ApplyCABundles,
			common.CopyConfigMap(pipelinesAsCodeCM, pac.Spec.Settings),
			occommon.UpdateServiceMonitorTargetNamespace(pac.Spec.TargetNamespace),
		}

		allTfs := append(tfs, extension.Transformers(pac)...)
		if err := common.Transform(ctx, &pacManifest, pac, allTfs...); err != nil {
			return &mf.Manifest{}, err
		}

		// additional options transformer
		// always execute as last transformer, so that the values in options will be final update values on the manifests
		if err := common.ExecuteAdditionalOptionsTransformer(ctx, &pacManifest, pac.Spec.GetTargetNamespace(), pac.Spec.Options); err != nil {
			return &mf.Manifest{}, err
		}

		return &pacManifest, nil
	}
}

// additional pac controller config
func additionalControllerTransform(extension common.Extension) client.FilterAndTransform {
	return func(ctx context.Context, additionalPACManifest *mf.Manifest, comp v1alpha1.TektonComponent) (*mf.Manifest, error) {

		return additionalPACManifest, nil
	}
}

func additionalControllerTransformTest(ctx context.Context, extension common.Extension, additionalPACManifest *mf.Manifest, comp v1alpha1.TektonComponent, addPACControllerConfig *v1alpha1.AdditionalPACControllerConfig) (*mf.Manifest, error) {

	pac := comp.(*v1alpha1.OpenShiftPipelinesAsCode)
	tfs := []mf.Transformer{
		common.InjectOperandNameLabelOverwriteExisting(openshift.OperandOpenShiftPipelineAsCodeAdditionalController + addPACControllerConfig.Name),
		updateAdditionControllerConfigMap(addPACControllerConfig),
		updateAdditionControllerDeployment(addPACControllerConfig),
		updateAdditionControllerService(addPACControllerConfig),
		updateAdditionControllerServiceMonitor(addPACControllerConfig),
		updateAdditionControllerRoute(addPACControllerConfig),
	}

	allTfs := append(tfs, extension.Transformers(pac)...)
	if err := common.Transform(ctx, additionalPACManifest, pac, allTfs...); err != nil {
		return &mf.Manifest{}, err
	}

	return additionalPACManifest, nil

}

func updateAdditionControllerConfigMap(config *v1alpha1.AdditionalPACControllerConfig) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "ConfigMap" {
			return nil
		}

		u.SetName(config.ConfigMapName)

		// }
		// u.SetName(config.Name + additionalPACControllerSuffix + "-route")
		// err := unstructured.SetNestedField(u.Object, config.Name+additionalPACControllerSuffix, "spec", "to", "name")
		// if err != nil {
		// 	return err
		// }
		return nil
	}
}

func updateAdditionControllerDeployment(config *v1alpha1.AdditionalPACControllerConfig) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Deployment" {
			return nil
		}

		u.SetName(config.Name + additionalPACControllerSuffix)

		d := &appsv1.Deployment{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, d)
		if err != nil {
			return err
		}

		d.Spec.Template.Labels["app"] = config.Name + additionalPACControllerSuffix
		d.Spec.Template.Labels["app.kubernetes.io/name"] = config.Name + additionalPACControllerSuffix
		d.Spec.Selector.MatchLabels["app.kubernetes.io/name"] = config.Name + additionalPACControllerSuffix
		containerEnvs := d.Spec.Template.Spec.Containers[0].Env
		d.Spec.Template.Spec.Containers[0].Env = replaceEnvInDeployment(containerEnvs, config)
		unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(d)
		if err != nil {
			return err
		}
		u.SetUnstructuredContent(unstrObj)

		return nil
	}
}

func updateAdditionControllerServiceMonitor(config *v1alpha1.AdditionalPACControllerConfig) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "ServiceMonitor" {
			return nil
		}

		var err error
		u.SetName(config.Name + additionalPACControllerSuffix + "-servicemonitor")

		err = unstructured.SetNestedMap(u.Object, map[string]interface{}{
			"app": config.Name + additionalPACControllerSuffix,
		}, "spec", "selector", "matchLabels")
		if err != nil {
			return err
		}

		err = unstructured.SetNestedStringSlice(u.Object, []string{"openshift-pipelines"},
			"spec", "namespaceSelector", "matchNames")
		if err != nil {
			return err
		}
		return nil
	}
}

func updateAdditionControllerService(config *v1alpha1.AdditionalPACControllerConfig) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Service" {
			return nil
		}
		var err error
		u.SetName(config.Name + additionalPACControllerSuffix)

		err = unstructured.SetNestedMap(u.Object, map[string]interface{}{
			"app.kubernetes.io/name": config.Name + additionalPACControllerSuffix,
		}, "spec", "selector")
		if err != nil {
			return err
		}
		err = unstructured.SetNestedMap(u.Object, map[string]interface{}{
			"app": config.Name,
		}, "metadata", "labels")
		if err != nil {
			return err
		}
		err = unstructured.SetNestedMap(u.Object, map[string]interface{}{
			"app.kubernetes.io/name": config.Name + additionalPACControllerSuffix,
		}, "metadata", "labels")
		if err != nil {
			return err
		}

		return nil
	}
}

func updateAdditionControllerRoute(config *v1alpha1.AdditionalPACControllerConfig) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Route" {
			return nil
		}
		u.SetName(config.Name + additionalPACControllerSuffix + "-route")
		err := unstructured.SetNestedField(u.Object, config.Name+additionalPACControllerSuffix, "spec", "to", "name")
		if err != nil {
			return err
		}
		return nil
	}
}

func replaceEnvInDeployment(envs []corev1.EnvVar, envInfo *v1alpha1.AdditionalPACControllerConfig) []corev1.EnvVar {
	for i, e := range envs {
		if e.Name == "PAC_CONTROLLER_CONFIGMAP" && envInfo.ConfigMapName == "" {
			envs[i].Value = envInfo.Name + additionalPACControllerSuffix + "config"
		}
		if e.Name == "PAC_CONTROLLER_CONFIGMAP" && envInfo.ConfigMapName != "" {
			envs[i].Value = envInfo.ConfigMapName
		}
		if e.Name == "PAC_CONTROLLER_SECRET" && envInfo.SecretsName != "" {
			envs[i].Value = envInfo.SecretsName
		}
		if e.Name == "PAC_CONTROLLER_LABEL" && envInfo.Name != "" {
			envs[i].Value = envInfo.Name + additionalPACControllerSuffix
		}
	}
	return envs
}
