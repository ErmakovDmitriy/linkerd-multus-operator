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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/constants"
	netattachclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

// AttachDefinitionReconciler reconciles a AttachDefinition object
type MultusNetAttachDefinitionReconciler struct {
	client.Client
	Scheme                 *runtime.Scheme
	InstanceName           string
	AttachReconcileTrigger func(ctx context.Context, req reconcile.Request) (reconcile.Result, error)
}

//+kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AttachDefinition object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *MultusNetAttachDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var logger = log.FromContext(ctx).WithValues("NetworkAttachmentDefinition", req.NamespacedName)

	logger.Info("Received reconcile event, run AttachDefinition reconcile")

	res, err := r.AttachReconcileTrigger(ctx, req)
	if err != nil {
		logger.Error(err, "can not reconcile AttachDefinition")

		return res, err
	}

	return res, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MultusNetAttachDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netattachclient.NetworkAttachmentDefinition{}).
		Named("MultusNetAttachDefinitionReconciler").
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
