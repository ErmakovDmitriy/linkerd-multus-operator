package webhooks

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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/annotate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=attachdefinition.cni.linkerd.io,admissionReviewVersions=v1,sideEffects=None

//+kubebuilder:rbac:groups="",resources=pods,verbs=get;versions=v1
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;versions=v1

type PodAnnotator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (a *PodAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := a.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	var isAnnotationRequested bool

	podAnnotation, ok := pod.Annotations[constants.LinkerdInjectAnnotation]
	if ok && (podAnnotation == "enabled" || podAnnotation == "ingress") {
		isAnnotationRequested = true
	} else {
		// Check Namespace.
		var (
			namespace    = &corev1.Namespace{}
			namespaceRef = client.ObjectKey{Name: pod.Namespace}
		)

		if err := a.Client.Get(ctx, namespaceRef, namespace); err != nil {
			if errors.IsNotFound(err) {
				return admission.Errored(http.StatusNotFound,
					fmt.Errorf("can not get Pod Namespace %s: %w", namespaceRef.Name, err))
			}

			return admission.Errored(http.StatusInternalServerError, err)
		}

		namespaceAnnot, ok := namespace.Annotations[constants.LinkerdInjectAnnotation]
		if ok && (namespaceAnnot == "enabled" || namespaceAnnot == "ingress") {
			isAnnotationRequested = true
		}
	}

	if !isAnnotationRequested {
		return admission.Allowed("No Multus annotation required")
	}

	// Check if Multus NetworkAttachDefinition is in the Pod's namespace.
	var (
		multus    = &netattachv1.NetworkAttachmentDefinition{}
		multusRef = client.ObjectKey{Namespace: pod.Namespace, Name: constants.LinkerdCNINetworkAttachmentDefinitionName}
	)

	if err := a.Client.Get(ctx, multusRef, multus); err != nil {
		if errors.IsNotFound(err) {
			return admission.Allowed("Multus NetworkAttachDefinition " + constants.LinkerdCNINetworkAttachmentDefinitionName + "is not in Namespace")
		}

		return admission.Errored(http.StatusInternalServerError,
			fmt.Errorf("can not get Multus NetworkAttachDefinition %s: %w", multusRef.String(), err))
	}

	// Patch.
	currentNetworks, ok := pod.Annotations[constants.MultusNetworkAttachAnnotation]
	if ok {
		pod.Annotations[constants.MultusNetworkAttachAnnotation] = currentNetworks + "," + constants.LinkerdCNINetworkAttachmentDefinitionName
	} else {
		pod.Annotations[constants.MultusNetworkAttachAnnotation] = constants.LinkerdCNINetworkAttachmentDefinitionName
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func (a *PodAnnotator) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
