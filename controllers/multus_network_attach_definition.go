package controllers

import (
	"encoding/json"

	"github.com/containernetworking/cni/pkg/types"
	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LinkerdCNIVersion = "0.3.0"
	LinkerdCNIName    = "linkerd-cni"
	LinkerdCNIType    = "linkerd-cni"
)

// // var (
// // 	LinkerdCNIPolicy = Policy{
// // 		Type:         "k8s",
// // 		K8sAPIRoot:   "https://__KUBERNETES_SERVICE_HOST__:__KUBERNETES_SERVICE_PORT__",
// // 		K8sAuthToken: "__SERVICEACCOUNT_TOKEN__",
// // 	}
// // 	LinkerdCNIKubernetes = Kubernetes{
// // 		K8sAPIRoot: "https://__KUBERNETES_SERVICE_HOST__:__KUBERNETES_SERVICE_PORT__",
// // 		Kubeconfig: "__KUBECONFIG_FILEPATH__",
// // 	}
// // )

// ProxyInit is the configuration for the proxy-init binary.
type ProxyInit struct {
	IncomingProxyPort     int      `json:"incoming-proxy-port"`
	OutgoingProxyPort     int      `json:"outgoing-proxy-port"`
	ProxyUID              int      `json:"proxy-uid"`
	PortsToRedirect       []int    `json:"ports-to-redirect"`
	InboundPortsToIgnore  []string `json:"inbound-ports-to-ignore"`
	OutboundPortsToIgnore []string `json:"outbound-ports-to-ignore"`
	Simulate              bool     `json:"simulate"`
	UseWaitFlag           bool     `json:"use-wait-flag"`
}

// Kubernetes a K8s specific struct to hold config.
type Kubernetes struct {
	K8sAPIRoot string `json:"k8s_api_root"`
	Kubeconfig string `json:"kubeconfig"`
}

// Policy a K8s struct to hold policy. Not used as I understand.
type Policy struct {
	Type         string `json:"type"`
	K8sAPIRoot   string `json:"k8s_api_root"`
	K8sAuthToken string `json:"k8s_auth_token"`
}

// CNIPluginConf is whatever JSON is passed via stdin.
type CNIPluginConf struct {
	types.NetConf

	LogLevel string `json:"log_level"`

	Linkerd ProxyInit `json:"linkerd"`

	Kubernetes Kubernetes `json:"kubernetes"`
	Policy     Policy     `json:"policy"`
}

func newCNIPluginConf() *CNIPluginConf {
	return &CNIPluginConf{
		NetConf: types.NetConf{
			CNIVersion: LinkerdCNIVersion,
			Name:       LinkerdCNIName,
			Type:       LinkerdCNIType,
		},
	}
}

func newMultusNetworkAttachDefinition(
	multusRef client.ObjectKey, config *CNIPluginConf) (*netattachv1.NetworkAttachmentDefinition, error) {
	var multusNetAttach = &netattachv1.NetworkAttachmentDefinition{
		TypeMeta: v1.TypeMeta{
			Kind:       "NetworkAttachmentDefinition",
			APIVersion: "k8s.cni.cncf.io/v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      multusRef.Name,
			Namespace: multusRef.Namespace,
			// OwnerReferences: []v1.OwnerReference{
			// 	{
			// 		// APIVersion: ,

			// 	},
			// },
		},
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	multusNetAttach.Spec.Config = string(cfg)

	return multusNetAttach, nil
}
