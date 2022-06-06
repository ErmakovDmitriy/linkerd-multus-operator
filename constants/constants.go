package constants

const (
	LinkerdCNIVersion = "0.3.0"
	LinkerdCNIName    = "linkerd-cni"
	LinkerdCNIType    = "linkerd-cni"

	LinkerdCNINetworkAttachmentDefinitionName = "linkerd-cni"

	LinkerdInjectAnnotation = "linkerd.io/inject"
)

const (
	MultusNetworkAttachmentDefinitionAPIVersion   = "k8s.cni.cncf.io/v1"
	MultusNetworkAttachmentDefinitionResourceKind = "NetworkAttachmentDefinition"
	MultusNetworkAttachAnnotation                 = "k8s.v1.cni.cncf.io/networks"
)
