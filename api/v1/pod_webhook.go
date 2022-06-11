/*
Copyright 2022.

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

package v1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/constants"
	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/controllers"
	"github.com/go-logr/logr"
	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//nolint:lll
//+kubebuilder:webhook:path=/annotate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create;update,versions=v1,name=attachdefinition.cni.linkerd.io,admissionReviewVersions=v1,sideEffects=None
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;versions=v1
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;versions=v1
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;versions=v1

const (
	LinkerdCNIAnnotationEnabled = "enabled"
	LinkerdCNIAnnotationIngress = "ingress"
)

type PodAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle - main Multus annotator handler.
// nolint:gocritic // hugeParam - admission.Request is passed not as a pointer to conform to admission.Handler interface.
func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues("request_namespace", req.Namespace, "request_name", req.Name)

	logger.Info("Received admission request", "request", req)

	pod := &corev1.Pod{}
	err := a.decoder.Decode(req, pod)
	if err != nil {
		logger.Error(err, "can not decode v1.Pod")

		return admission.Errored(http.StatusBadRequest, err)
	}

	logger.Info("Loaded Pod info", "pod_generate_name", pod.GenerateName)

	var isMultusAnnotationRequested = isCNIRequestedByPod(logger, pod)

	// Check Namespace.
	if !isMultusAnnotationRequested {
		isMultusAnnotationRequested, err = isCNIRequestedByNamespace(ctx, req.Namespace, logger, a.Client)
		if err != nil {
			return errorToResponse(err)
		}
	}

	if !isMultusAnnotationRequested {
		logger.Info(
			"Multus NetworkAttachmentDefinition is not required as neither Pod nor Namespace requested Linkerd proxy inject")

		return admission.Allowed("No Multus annotation requested")
	}

	// Check if Multus NetworkAttachDefinition is in the Pod's namespace.
	multus, err := getMultus(ctx, logger, a.Client, req.Namespace)
	if err != nil {
		return errorToResponse(err)
	}

	// Patch ProxyUID from the Multus definition if necessary.
	// ToDo: In the future, I think, this should be done by the Proxy Inject web hook and
	// removed from this controller as the proxy inject will be able to
	// set other options such as resource requests, limits, ports etc.
	if _, ok := pod.Annotations[constants.LinkerdProxyUIDAnnotation]; !ok {
		var linkerdCNIConfig = &controllers.CNIPluginConf{}

		if err = json.Unmarshal([]byte(multus.Spec.Config), linkerdCNIConfig); err != nil {
			logger.Error(err, "can not Unmarshal Multus.Spec.Config to CNIPluginConf")

			return admission.Errored(
				http.StatusInternalServerError,
				fmt.Errorf("can not Unmarshal Multus.Spec.Config to CNIPluginConf, multus=%s/%s, error=%w",
					multus.Namespace, multus.Name, err))
		}

		pod.Annotations[constants.LinkerdProxyUIDAnnotation] = strconv.Itoa(linkerdCNIConfig.Linkerd.ProxyUID)
	}

	// Patch NetworkAttachmentDefinitions list.
	logger.Info("Pod network annotation is",
		constants.MultusNetworkAttachAnnotation, pod.Annotations[constants.MultusNetworkAttachAnnotation])
	pod = patchPodNetworks(logger, pod)
	logger.Info("Patched Pod annotation is",
		constants.MultusNetworkAttachAnnotation, pod.Annotations[constants.MultusNetworkAttachAnnotation])

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error(err, "can not json.Marshal patched Pod definition")

		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (a *PodAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}

// isCNIRequestedByPod - checks if a Pod contains linkerd inject annotation.
func isCNIRequestedByPod(logger logr.Logger, pod *corev1.Pod) bool {
	podAnnotation, ok := pod.Annotations[constants.LinkerdInjectAnnotation]
	if ok && (podAnnotation == LinkerdCNIAnnotationEnabled || podAnnotation == LinkerdCNIAnnotationIngress) {
		logger.Info("Pod contains inject annotation", constants.LinkerdInjectAnnotation, podAnnotation)

		return true
	}

	return false
}

// isCNIRequestedByNamespace - if a Namespace contains linkerd inject annotation.
func isCNIRequestedByNamespace(ctx context.Context, namespaceName string, logger logr.Logger, apiClient client.Client) (bool, error) {
	// Check Namespace.
	var (
		namespaceRef = client.ObjectKey{Name: namespaceName}
		namespace    = &corev1.Namespace{}
	)

	logger.Info("Checking Namespace annotation", "namespaceRef", namespaceRef.Name)

	if err := apiClient.Get(ctx, namespaceRef, namespace); err != nil {
		logger.Error(err, "can not get namespace", "namespaceRef", namespaceRef.Name)

		return false, err
	}

	namespaceAnnot, ok := namespace.Annotations[constants.LinkerdInjectAnnotation]
	if ok && (namespaceAnnot == "enabled" || namespaceAnnot == "ingress") {
		logger.Info("Namespace contains inject annotation", constants.LinkerdInjectAnnotation, namespaceAnnot)

		return true, nil
	}

	return false, nil
}

func errorToResponse(err error) admission.Response {
	if status := apierrors.APIStatus(nil); errors.As(err, &status) {
		return admission.Errored(status.Status().Code, err)
	}

	return admission.Errored(http.StatusInternalServerError, err)
}

// getMultus - loads a Multus NetworkAttachmentDefinition from K8s API.
func getMultus(ctx context.Context, logger logr.Logger, apiClient client.Client,
	namespaceName string) (*netattachv1.NetworkAttachmentDefinition, error) {
	// Check if Multus NetworkAttachDefinition is in the Pod's namespace.
	var (
		multus    = &netattachv1.NetworkAttachmentDefinition{}
		multusRef = client.ObjectKey{
			Namespace: namespaceName,
			Name:      constants.LinkerdCNINetworkAttachmentDefinitionName,
		}
	)

	logger.Info("Trying to get MultusNetworkAttachmentDefinition", "multusRef", multusRef.String())

	if err := apiClient.Get(ctx, multusRef, multus); err != nil {
		if apierrors.IsNotFound(err) {
			//nolint:stylecheck
			var errWrap = fmt.Errorf(
				"Multus NetworkAttachDefinition %s is not found: %w",
				multusRef.String(), err)
			logger.Error(errWrap, "Not found")

			return nil, errWrap
		}

		var errWrap = fmt.Errorf(
			"can not get Multus NetworkAttachDefinition, multusRef=%s, error=%w",
			multusRef.String(), err)

		logger.Error(errWrap, "Get error")

		return nil, errWrap
	}

	return multus, nil
}

// patchPodNetworks - adds Linkerd CNI network to NetworkAttachmentDefinitions list of a Pod.
func patchPodNetworks(logger logr.Logger, pod *corev1.Pod) *corev1.Pod {
	currentNetworks, ok := pod.Annotations[constants.MultusNetworkAttachAnnotation]

	logger.Info("Pod annotation is", constants.MultusNetworkAttachAnnotation, currentNetworks)

	if ok {
		// Check that there is no the Linkerd CNI annotation already.
		nets := strings.Split(currentNetworks, ",")

		var isAnnotationNeeded = true
		for _, net := range nets {
			if net == constants.LinkerdCNINetworkAttachmentDefinitionName {
				isAnnotationNeeded = false
				break
			}
		}

		if isAnnotationNeeded {
			pod.Annotations[constants.MultusNetworkAttachAnnotation] = currentNetworks + "," + constants.LinkerdCNINetworkAttachmentDefinitionName
		}
	} else {
		pod.Annotations[constants.MultusNetworkAttachAnnotation] = constants.LinkerdCNINetworkAttachmentDefinitionName
	}

	return pod
}
