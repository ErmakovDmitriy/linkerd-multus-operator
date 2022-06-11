package controllers

import (
	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/constants"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// getEventFilter returns a static filter which omits events for all objects which do not have
// a hard-coded constants.LinkerdCNINetworkAttachmentDefinitionName as its name.
func getEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(ce event.CreateEvent) bool {
			return ce.Object.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
		},
		UpdateFunc: func(ue event.UpdateEvent) bool {
			return (ue.ObjectNew.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName ||
				ue.ObjectOld.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName)
		},
		DeleteFunc: func(de event.DeleteEvent) bool {
			return de.Object.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
		},
		GenericFunc: func(ge event.GenericEvent) bool {
			return ge.Object.GetName() == constants.LinkerdCNINetworkAttachmentDefinitionName
		},
	}
}
