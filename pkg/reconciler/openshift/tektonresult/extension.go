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

package tektonresult

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	operatorclient "github.com/tektoncd/operator/pkg/client/injection/client"
	"github.com/tektoncd/operator/pkg/reconciler/common"
	"github.com/tektoncd/operator/pkg/reconciler/kubernetes/tektoninstallerset/client"
	occommon "github.com/tektoncd/operator/pkg/reconciler/openshift/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/logging"
)

const (
	// manifests console plugin yaml directory location
	routeRBACYamlDirectory    = "static/tekton-results/route-rbac"
	routeIngressYamlDirectory = "static/tekton-results/route-ingress"
	internalDBYamlDirectory   = "static/tekton-results/internal-db"
	logsRBACYamlDirectory     = "static/tekton-results/logs-rbac"
	deploymentAPI             = "tekton-results-api"
	serviceAPI                = "tekton-results-api-service"
	secretAPITLS              = "tekton-results-tls"
	apiContainerName          = "api"
	boundSAVolume             = "bound-sa-token"
	boundSAPath               = "/var/run/secrets/openshift/serviceaccount"
	lokiStackTLSCAEnvVar      = "LOGGING_PLUGIN_CA_CERT"
	tektonResultWatcherName   = "tekton-results-watcher"
)

func OpenShiftExtension(ctx context.Context) common.Extension {
	logger := logging.FromContext(ctx)

	version := os.Getenv(v1alpha1.VersionEnvKey)
	if version == "" {
		logger.Fatal("Failed to find version from env")
	}

	routeManifest, err := getRouteManifest()
	if err != nil {
		logger.Fatalf("Failed to fetch route rbac static manifest: %v", err)
	}

	internalDBManifest, err := getDBManifest()
	if err != nil {
		logger.Fatalf("Failed to fetch internal db static manifest: %v", err)
	}

	logsRBACManifest, err := getloggingRBACManifest()
	if err != nil {
		logger.Fatalf("Failed to fetch logs RBAC manifest: %v", err)
	}

	routeIngressManifest, err := getRouteIngressManifest()
	if err != nil {
		logger.Fatalf("Failed to fetch route ingress manifest: %v", err)
	}

	ext := &openshiftExtension{
		installerSetClient: client.NewInstallerSetClient(operatorclient.Get(ctx).OperatorV1alpha1().TektonInstallerSets(),
			version, "results-ext", v1alpha1.KindTektonResult, nil),
		internalDBManifest:   internalDBManifest,
		routeManifest:        routeManifest,
		logsRBACManifest:     logsRBACManifest,
		routeIngressManifest: routeIngressManifest,
	}
	return ext
}

type openshiftExtension struct {
	installerSetClient   *client.InstallerSetClient
	routeManifest        *mf.Manifest
	internalDBManifest   *mf.Manifest
	logsRBACManifest     *mf.Manifest
	routeIngressManifest *mf.Manifest
	removePreset         bool
}

func (oe openshiftExtension) Transformers(comp v1alpha1.TektonComponent) []mf.Transformer {
	instance := comp.(*v1alpha1.TektonResult)

	return []mf.Transformer{
		occommon.RemoveRunAsUser(),
		occommon.RemoveRunAsGroup(),
		occommon.ApplyCABundlesToDeployment,
		occommon.RemoveRunAsUserForStatefulSet(tektonResultWatcherName),
		occommon.RemoveRunAsGroupForStatefulSet(tektonResultWatcherName),
		occommon.ApplyCABundlesForStatefulSet(tektonResultWatcherName),
		injectBoundSAToken(instance.Spec.ResultsAPIProperties),
		injectLokiStackTLSCACert(instance.Spec.LokiStackProperties),
		injectResultsAPIServiceCACert(instance.Spec.ResultsAPIProperties),
	}
}

func (oe *openshiftExtension) PreReconcile(ctx context.Context, tc v1alpha1.TektonComponent) error {
	result := tc.(*v1alpha1.TektonResult)
	mf := mf.Manifest{}

	if !result.Spec.IsExternalDB {
		mf = *oe.internalDBManifest
		oe.removePreset = true
	}
	if result.Spec.IsExternalDB && oe.removePreset {
		if err := oe.installerSetClient.CleanupPreSet(ctx); err != nil {
			return err
		}
		oe.removePreset = false
	}
	if (result.Spec.LokiStackName != "" && result.Spec.LokiStackNamespace != "") ||
		strings.EqualFold(result.Spec.LogsType, "LOKI") {
		mf = mf.Append(*oe.logsRBACManifest)
	}

	return oe.installerSetClient.PreSet(ctx, tc, &mf, filterAndTransform())
}

func (oe openshiftExtension) PostReconcile(ctx context.Context, tc v1alpha1.TektonComponent) error {
	result := tc.(*v1alpha1.TektonResult)
	mf := *oe.routeManifest

	// Conditionally add route and ingress manifests based on configuration
	if shouldEnableRoute(result) || shouldEnableIngress(result) {
		mf = mf.Append(*oe.routeIngressManifest)
	}

	return oe.installerSetClient.PostSet(ctx, tc, &mf, filterAndTransformRouteIngress())
}

func (oe openshiftExtension) Finalize(ctx context.Context, tc v1alpha1.TektonComponent) error {
	if err := oe.installerSetClient.CleanupPostSet(ctx); err != nil {
		return err
	}
	if err := oe.installerSetClient.CleanupPreSet(ctx); err != nil {
		return err
	}
	return nil
}

func getRouteManifest() (*mf.Manifest, error) {
	manifest := &mf.Manifest{}
	resultsRbac := filepath.Join(common.ComponentBaseDir(), routeRBACYamlDirectory)
	if err := common.AppendManifest(manifest, resultsRbac); err != nil {
		return nil, err
	}
	return manifest, nil
}

func getDBManifest() (*mf.Manifest, error) {
	manifest := &mf.Manifest{}
	internalDB := filepath.Join(common.ComponentBaseDir(), internalDBYamlDirectory)
	if err := common.AppendManifest(manifest, internalDB); err != nil {
		return nil, err
	}
	return manifest, nil
}

// function to add fine grained access control to results api if results config specifies that
// pipeline logs are managed by OpenShift Logging with OpenShift logging and OpenShift loki operators
func getloggingRBACManifest() (*mf.Manifest, error) {
	manifest := &mf.Manifest{}
	logsRbac := filepath.Join(common.ComponentBaseDir(), logsRBACYamlDirectory)
	if err := common.AppendManifest(manifest, logsRbac); err != nil {
		return nil, err
	}
	return manifest, nil
}

func getRouteIngressManifest() (*mf.Manifest, error) {
	manifest := &mf.Manifest{}
	routeIngress := filepath.Join(common.ComponentBaseDir(), routeIngressYamlDirectory)
	if err := common.AppendManifest(manifest, routeIngress); err != nil {
		return nil, err
	}
	return manifest, nil
}

func filterAndTransform() client.FilterAndTransform {
	return func(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.TektonComponent) (*mf.Manifest, error) {
		resultImgs := common.ToLowerCaseKeys(common.ImagesFromEnv(common.ResultsImagePrefix))

		extra := []mf.Transformer{
			common.InjectOperandNameLabelOverwriteExisting(v1alpha1.OperandTektoncdResults),
			common.ApplyProxySettings,
			common.AddStatefulSetRestrictedPSA(),
			common.DeploymentImages(resultImgs),
			common.StatefulSetImages(resultImgs),
		}

		if err := common.Transform(ctx, manifest, comp, extra...); err != nil {
			return nil, err
		}
		return manifest, nil
	}
}

func injectResultsAPIServiceCACert(props v1alpha1.ResultsAPIProperties) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Service" || u.GetName() != serviceAPI {
			return nil
		}

		s := &corev1.Service{}
		err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, s)
		if err != nil {
			return err
		}

		annotations := s.Annotations
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["service.beta.openshift.io/serving-cert-secret-name"] = secretAPITLS
		s.SetAnnotations(annotations)

		uObj, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(s)
		if err != nil {
			return err
		}
		u.SetUnstructuredContent(uObj)
		return nil
	}
}

// injectBoundSAToken adds a sa token projected volume to the Results Deployment
func injectBoundSAToken(props v1alpha1.ResultsAPIProperties) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if props.LogsAPI == nil || !*props.LogsAPI ||
			u.GetKind() != "Deployment" || u.GetName() != deploymentAPI {
			return nil
		}

		d := &appsv1.Deployment{}
		err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, d)
		if err != nil {
			return err
		}

		// find the matching container and add env and secret name object
		for i, container := range d.Spec.Template.Spec.Containers {
			if container.Name != apiContainerName {
				continue
			}
			add := true
			vol := corev1.Volume{
				Name: boundSAVolume,
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources: []corev1.VolumeProjection{
							{
								ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
									Audience: "openshift",
									Path:     "token",
								},
							},
						},
					},
				},
			}
			for k := 0; k < len(d.Spec.Template.Spec.Volumes); k++ {
				if d.Spec.Template.Spec.Volumes[k].Name == boundSAVolume {
					d.Spec.Template.Spec.Volumes[k] = vol
					add = false
				}
			}
			if add {
				d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, vol)
			}

			volMount := corev1.VolumeMount{
				Name:      boundSAVolume,
				MountPath: boundSAPath,
			}

			add = true
			for k := 0; k < len(d.Spec.Template.Spec.Containers[i].VolumeMounts); k++ {
				if d.Spec.Template.Spec.Containers[i].VolumeMounts[k].Name == boundSAVolume {
					d.Spec.Template.Spec.Containers[i].VolumeMounts[k] = volMount
					add = false
				}
			}
			if add {
				d.Spec.Template.Spec.Containers[i].VolumeMounts = append(
					d.Spec.Template.Spec.Containers[i].VolumeMounts, volMount)
			}

			break
		}

		uObj, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(d)
		if err != nil {
			return err
		}
		u.SetUnstructuredContent(uObj)
		return nil
	}
}

// injectLokiStackTLSCACert adds a tls ca cert environment variable to the Results Deployment
// If the env variable already exists, it will be overwritten
func injectLokiStackTLSCACert(prop v1alpha1.LokiStackProperties) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if prop.LokiStackNamespace == "" || prop.LokiStackName == "" ||
			u.GetKind() != "Deployment" || u.GetName() != deploymentAPI {
			return nil
		}

		d := &appsv1.Deployment{}
		err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(u.Object, d)
		if err != nil {
			return err
		}

		// find the matching container and add env and secret name object
		for i, container := range d.Spec.Template.Spec.Containers {
			if container.Name != apiContainerName {
				continue
			}
			add := true
			env := corev1.EnvVar{
				Name: lokiStackTLSCAEnvVar,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "openshift-service-ca.crt",
						},
						Key: "service-ca.crt",
					},
				},
			}

			// Check if the env variable already exists in the container
			// If it does, overwrite it
			for k := 0; k < len(d.Spec.Template.Spec.Containers[i].Env); k++ {
				if d.Spec.Template.Spec.Containers[i].Env[k].Name == lokiStackTLSCAEnvVar {
					d.Spec.Template.Spec.Containers[i].Env[k] = env
					add = false
				}
			}

			// If it doesn't exist, add it
			if add {
				d.Spec.Template.Spec.Containers[i].Env = append(
					d.Spec.Template.Spec.Containers[i].Env, env)
			}

			break
		}

		uObj, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(d)
		if err != nil {
			return err
		}
		u.SetUnstructuredContent(uObj)
		return nil
	}
}

// shouldEnableRoute determines if route should be enabled for results API
func shouldEnableRoute(result *v1alpha1.TektonResult) bool {
	// Default to false if not explicitly set
	if result.Spec.RouteEnabled == nil {
		return false
	}
	return *result.Spec.RouteEnabled
}

// shouldEnableIngress determines if ingress should be enabled for results API
func shouldEnableIngress(result *v1alpha1.TektonResult) bool {
	// Default to false if not explicitly set
	if result.Spec.IngressEnabled == nil {
		return false
	}
	return *result.Spec.IngressEnabled
}

// filterAndTransformRouteIngress provides filtering and transformation for route/ingress resources
func filterAndTransformRouteIngress() client.FilterAndTransform {
	return func(ctx context.Context, manifest *mf.Manifest, comp v1alpha1.TektonComponent) (*mf.Manifest, error) {
		result := comp.(*v1alpha1.TektonResult)
		resultImgs := common.ToLowerCaseKeys(common.ImagesFromEnv(common.ResultsImagePrefix))

		extra := []mf.Transformer{
			common.InjectOperandNameLabelOverwriteExisting(v1alpha1.OperandTektoncdResults),
			common.ApplyProxySettings,
			common.DeploymentImages(resultImgs),
			common.StatefulSetImages(resultImgs),
			configureRoute(result),
			configureIngress(result),
		}

		// Filter manifests based on configuration
		filteredManifest := *manifest

		// Filter out route if not enabled
		if !shouldEnableRoute(result) {
			filteredManifest = filteredManifest.Filter(mf.Not(mf.ByKind("Route")))
		}

		// Filter out ingress if not enabled
		if !shouldEnableIngress(result) {
			filteredManifest = filteredManifest.Filter(mf.Not(mf.ByKind("Ingress")))
		}

		if err := common.Transform(ctx, &filteredManifest, comp, extra...); err != nil {
			return nil, err
		}
		return &filteredManifest, nil
	}
}

// configureRoute transformer to configure route based on TektonResult spec
func configureRoute(result *v1alpha1.TektonResult) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Route" || u.GetName() != serviceAPI {
			return nil
		}

		// Apply custom host if specified
		if result.Spec.RouteHost != "" {
			if err := unstructured.SetNestedField(u.Object, result.Spec.RouteHost, "spec", "host"); err != nil {
				return err
			}
		}

		// Apply custom path if specified
		if result.Spec.RoutePath != "" {
			if err := unstructured.SetNestedField(u.Object, result.Spec.RoutePath, "spec", "path"); err != nil {
				return err
			}
		}

		// Apply custom TLS termination if specified
		if result.Spec.RouteTLSTermination != "" {
			if err := unstructured.SetNestedField(u.Object, result.Spec.RouteTLSTermination, "spec", "tls", "termination"); err != nil {
				return err
			}
		}

		return nil
	}
}

// configureIngress transformer to configure ingress based on TektonResult spec
func configureIngress(result *v1alpha1.TektonResult) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if u.GetKind() != "Ingress" || u.GetName() != serviceAPI {
			return nil
		}

		// Apply custom host if specified
		if result.Spec.IngressHost != "" {
			// Get the rules array, modify it, then set it back
			rules, found, err := unstructured.NestedSlice(u.Object, "spec", "rules")
			if err != nil || !found || len(rules) == 0 {
				return err
			}
			rule := rules[0].(map[string]interface{})
			rule["host"] = result.Spec.IngressHost
			rules[0] = rule
			if err := unstructured.SetNestedSlice(u.Object, rules, "spec", "rules"); err != nil {
				return err
			}

			// Also set it in TLS section if TLS is enabled
			if result.Spec.IngressTLS != nil && *result.Spec.IngressTLS {
				tls, found, err := unstructured.NestedSlice(u.Object, "spec", "tls")
				if err != nil || !found || len(tls) == 0 {
					return err
				}
				tlsEntry := tls[0].(map[string]interface{})
				hosts := []interface{}{result.Spec.IngressHost}
				tlsEntry["hosts"] = hosts
				tls[0] = tlsEntry
				if err := unstructured.SetNestedSlice(u.Object, tls, "spec", "tls"); err != nil {
					return err
				}
			}
		}

		// Apply custom path if specified
		if result.Spec.IngressPath != "" {
			// Get the rules array, modify the path, then set it back
			rules, found, err := unstructured.NestedSlice(u.Object, "spec", "rules")
			if err != nil || !found || len(rules) == 0 {
				return err
			}
			rule := rules[0].(map[string]interface{})
			http := rule["http"].(map[string]interface{})
			paths := http["paths"].([]interface{})
			path := paths[0].(map[string]interface{})
			path["path"] = result.Spec.IngressPath
			paths[0] = path
			http["paths"] = paths
			rule["http"] = http
			rules[0] = rule
			if err := unstructured.SetNestedSlice(u.Object, rules, "spec", "rules"); err != nil {
				return err
			}
		}

		// Remove TLS section if TLS is disabled
		if result.Spec.IngressTLS != nil && !*result.Spec.IngressTLS {
			unstructured.RemoveNestedField(u.Object, "spec", "tls")
		}

		return nil
	}
}
