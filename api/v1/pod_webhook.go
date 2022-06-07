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
	"fmt"
	"net/http"

	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/constants"
	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/annotate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create;update,versions=v1,name=attachdefinition.cni.linkerd.io,admissionReviewVersions=v1,sideEffects=None
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;versions=v1
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;versions=v1
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;versions=v1

type PodAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues("request_namespace", req.Namespace, "request_name", req.Name)

	logger.Info("Received admission request", "request", req)

	pod := &corev1.Pod{}
	err := a.decoder.Decode(req, pod)
	if err != nil {
		logger.Error(err, "can not decode v1.Pod")

		return admission.Errored(http.StatusBadRequest, err)
	}

	logger.Info("Loaded Pod info", "pod_namespace", req.Namespace, "pod_generate_name", pod.GenerateName)

	var isAnnotationRequested bool

	podAnnotation, ok := pod.Annotations[constants.LinkerdInjectAnnotation]
	if ok && (podAnnotation == "enabled" || podAnnotation == "ingress") {
		logger.Info("Pod contains inject annotation", constants.LinkerdInjectAnnotation, podAnnotation)

		isAnnotationRequested = true
	} else {
		// Check Namespace.
		var (
			namespace    = &corev1.Namespace{}
			namespaceRef = client.ObjectKey{Name: req.Namespace}
		)

		logger.Info("Checking Namespace annotation", "namespaceRef", namespaceRef.Name)

		if err := a.Client.Get(ctx, namespaceRef, namespace); err != nil {
			if errors.IsNotFound(err) {
				logger.Error(err, "can not get Pod Namespace", "namespaceRef", namespaceRef.Name)

				return admission.Errored(http.StatusNotFound,
					fmt.Errorf("can not get Pod Namespace, namespaceRef=%s, error=%w", namespaceRef.Name, err))
			}

			logger.Error(err, "Get Namespace error", "namespaceRef", namespaceRef.Name)

			return admission.Errored(http.StatusInternalServerError, err)
		}

		namespaceAnnot, ok := namespace.Annotations[constants.LinkerdInjectAnnotation]
		if ok && (namespaceAnnot == "enabled" || namespaceAnnot == "ingress") {
			logger.Info("Namespace contains inject annotation", constants.LinkerdInjectAnnotation, namespaceAnnot)

			isAnnotationRequested = true
		}
	}

	if !isAnnotationRequested {
		logger.Info("Multus NetworkAttachmentDefinition is not required as neither Pod nor Namespace require Linkerd proxy inject")

		return admission.Allowed("No Multus annotation required")
	}

	// Check if Multus NetworkAttachDefinition is in the Pod's namespace.
	var (
		multus    = &netattachv1.NetworkAttachmentDefinition{}
		multusRef = client.ObjectKey{Namespace: req.Namespace, Name: constants.LinkerdCNINetworkAttachmentDefinitionName}
	)

	logger.Info("Trying to get MultusNetworkAttachmentDefinition", "multusRef", multusRef.String())

	if err := a.Client.Get(ctx, multusRef, multus); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Multus NetworkAttachDefinition " + constants.LinkerdCNINetworkAttachmentDefinitionName + "is not in Namespace")

			return admission.Allowed("Multus NetworkAttachDefinition " + constants.LinkerdCNINetworkAttachmentDefinitionName + "is not in Namespace")
		}

		logger.Error(err, "Can not get Multus NetworkAttachmentDefinition", "multusRef=", multusRef.String())

		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("can not get Multus NetworkAttachDefinition, multusRef=%s, error=%w", multusRef.String(), err))
	}

	// Patch.
	currentNetworks, ok := pod.Annotations[constants.MultusNetworkAttachAnnotation]

	logger.Info("Pod annotation is", constants.MultusNetworkAttachAnnotation, currentNetworks)

	if ok {
		pod.Annotations[constants.MultusNetworkAttachAnnotation] = currentNetworks + "," + constants.LinkerdCNINetworkAttachmentDefinitionName
	} else {
		pod.Annotations[constants.MultusNetworkAttachAnnotation] = constants.LinkerdCNINetworkAttachmentDefinitionName
	}

	logger.Info("New Pod annotation is", constants.MultusNetworkAttachAnnotation, pod.Annotations[constants.MultusNetworkAttachAnnotation])

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
