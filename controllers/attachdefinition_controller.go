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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"errors"

	cniv1alpha1 "github.com/ErmakovDmitriy/linkerd-cni-attach-operator/api/v1alpha1"
	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/constants"
	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

var ErrCNIConfigMapKeyNotFound = errors.New("Linkerd CNI ConfigMap does not contain required key")

// AttachDefinitionReconciler reconciles a AttachDefinition object
type AttachDefinitionReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	InstanceName    string
	CNIKubeconfig   string
	CNIConfigMapRef CNIConfigMapRef
}

type CNIConfigMapRef struct {
	client.ObjectKey
	Key string
}

//+kubebuilder:rbac:groups=cni.linkerd.io,resources=attachdefinitions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cni.linkerd.io,resources=attachdefinitions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cni.linkerd.io,resources=attachdefinitions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AttachDefinition object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *AttachDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("AttachDefinition", req.NamespacedName)

	logger.Info("Received event")

	multusRef := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      constants.LinkerdCNINetworkAttachmentDefinitionName,
	}

	var linkerdAttach = &cniv1alpha1.AttachDefinition{}

	err := r.Get(ctx, req.NamespacedName, linkerdAttach)
	if err != nil {
		// Delete dependent resources - Multus NetworkAttachmentDefinition.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, r.deleteMultusNetAttach(ctx, multusRef)
		}

		return ctrl.Result{}, err
	}

	// Multus NetworkAttachmentDefinition is not requested - delete.
	if !linkerdAttach.Spec.CreateMultusNetworkAttachmentDefinition {
		logger.Info("createMultusNetworkAttachmentDefinition is false, delete NetworkAttachmentDefinition")

		return ctrl.Result{}, r.deleteMultusNetAttach(ctx, multusRef)
	}

	// Create/Update Multus NetworkAttachmentDefinition.

	// Load CNI Plugin configuration from a Linkerd CNI plugin ConfigMap.
	cniConfigDefault, err := r.getLinkerdCNIConfig(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Merge Linkerd CNI ConfigMap and linkerdAttach before further steps.
	var cniConfig = applyAttachDefinition(cniConfigDefault, linkerdAttach)

	var currentMultusNetAttach = &netattachv1.NetworkAttachmentDefinition{}

	if err := r.Get(ctx, multusRef, currentMultusNetAttach); err != nil {
		if apierrors.IsNotFound(err) {
			// Create.
			return ctrl.Result{}, r.createMultusNetAttach(ctx, multusRef, cniConfig)
		}

		return ctrl.Result{}, err
	}

	// Update.
	// Prepare required state.
	requiredMultusNetAttach, err := newMultusNetworkAttachDefinition(multusRef, cniConfig)
	if err != nil {
		logger.Error(err, "can not create expected NetworkAttachmentDefinition")

		return ctrl.Result{}, err
	}

	// Not very good comparison but will go for prototype.
	// ToDo: write a better comparison, maybe via json.Unmarshal.
	if currentMultusNetAttach.Spec.Config == requiredMultusNetAttach.Spec.Config {
		logger.Info("Current and required configurations are equal, nothing to do")

		return ctrl.Result{}, nil
	}

	currentMultusNetAttach.Spec.Config = requiredMultusNetAttach.Spec.Config

	logger.Info("Updating Multus NetworkAttachmentDefinition")

	if err := r.Update(ctx, currentMultusNetAttach); err != nil {
		logger.Error(err, "can not update NetworkAttachmentDefinition")

		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AttachDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cniv1alpha1.AttachDefinition{}).
		Named("AttachDefinitionReconciler").
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(ce event.CreateEvent) bool {
				return ce.Object.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
			},
			UpdateFunc: func(ue event.UpdateEvent) bool {
				return ue.ObjectNew.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName || ue.ObjectOld.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
			},
			DeleteFunc: func(de event.DeleteEvent) bool {
				return de.Object.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
			},
			GenericFunc: func(ge event.GenericEvent) bool {
				return ge.Object.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
			},
		}).
		Complete(r)
}

// deleteMultusNetAttach - deletes a Multus NetworkAttachmentDefinition.
func (r *AttachDefinitionReconciler) deleteMultusNetAttach(
	ctx context.Context, multusRef client.ObjectKey) error {
	logger := log.FromContext(ctx).WithValues(
		"k8s.cni.cncf.io/v1/NetworkAttachmentDefinition",
		multusRef.Namespace+"/"+multusRef.Name)

	logger.Info("Deleting Multus NetworkAttachmentDefinition")

	var multusNetAttach = &netattachv1.NetworkAttachmentDefinition{}

	// Get current Multus NetworkAttachmentDefinition state.

	if err := r.Get(ctx, multusRef, multusNetAttach); err != nil {
		// Already deleted, nothing to do.
		if apierrors.IsNotFound(err) {
			logger.Info("Object has been deleted earlier")

			return nil
		}

		logger.Error(err, "GET error")

		return err
	}

	if err := r.Delete(ctx, multusNetAttach); err != nil {
		// Already deleted, nothing to do.
		if apierrors.IsNotFound(err) {
			logger.Info("Object has been deleted earlier")

			return nil
		}

		logger.Error(err, "Delete error")

		return err
	}

	return nil
}

func (r *AttachDefinitionReconciler) createMultusNetAttach(ctx context.Context,
	multusRef client.ObjectKey, config *CNIPluginConf) error {
	logger := log.FromContext(ctx).WithValues(
		constants.MultusNetworkAttachmentDefinitionAPIVersion+"/"+constants.MultusNetworkAttachmentDefinitionResourceKind,
		multusRef.Namespace+"/"+multusRef.Name)

	logger.Info("Creating Multus NetworkAttachmentDefinition")

	multusNetAttach, err := newMultusNetworkAttachDefinition(multusRef, config)
	if err != nil {
		logger.Error(err, "can not Marshal CNI plugin configuration")

		return err
	}

	if err := r.Create(ctx, multusNetAttach); err != nil {
		logger.Error(err, "can not create Multus NetworkAttachmentDefinition")

		return err
	}

	return nil
}

// getLinkerdCNIConfig - loads CNI Plugin configuration from a Linkerd CNI plugin ConfigMap
// with patched KUBECONFIG path with the operator's provided value.
func (r *AttachDefinitionReconciler) getLinkerdCNIConfig(ctx context.Context) (*CNIPluginConf, error) {
	logger := log.FromContext(ctx).WithValues("v1/ConfigMap",
		r.CNIConfigMapRef.Namespace+"/"+r.CNIConfigMapRef.Name)

	var cniConfigMap = &corev1.ConfigMap{}
	if err := r.Get(ctx, r.CNIConfigMapRef.ObjectKey, cniConfigMap); err != nil {
		logger.Error(err, "can not get Linkerd CNI ConfigMap")

		return nil, err
	}

	cniRawConfig, ok := cniConfigMap.Data[r.CNIConfigMapRef.Key]
	if !ok {
		err := fmt.Errorf("%w: expected key %q", ErrCNIConfigMapKeyNotFound, r.CNIConfigMapRef.Key)

		logger.Error(err, "can not get Linkerd CNI ConfigMap")

		return nil, err
	}

	var cniConfig = newCNIPluginConf()

	if err := json.Unmarshal([]byte(cniRawConfig), cniConfig); err != nil {
		logger.Error(err, "Can not Unmarshal Linkerd CNI ConfigMap")

		return nil, err
	}

	// Set Kubeconfig.
	cniConfig.Kubernetes.Kubeconfig = r.CNIKubeconfig

	return cniConfig, nil
}
