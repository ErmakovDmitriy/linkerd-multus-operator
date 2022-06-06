package controllers

import (
	"encoding/json"
	"strconv"

	cniv1alpha1 "github.com/ErmakovDmitriy/linkerd-cni-attach-operator/api/v1alpha1"
	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/constants"
	"github.com/containernetworking/cni/pkg/types"
	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProxyInit is the configuration for the proxy-init binary.
type ProxyInit struct {
	IncomingProxyPort     int      `json:"incoming-proxy-port,omitempty"`
	OutgoingProxyPort     int      `json:"outgoing-proxy-port,omitempty"`
	ProxyUID              int      `json:"proxy-uid,omitempty"`
	PortsToRedirect       []int    `json:"ports-to-redirect,omitempty"`
	InboundPortsToIgnore  []string `json:"inbound-ports-to-ignore,omitempty"`
	OutboundPortsToIgnore []string `json:"outbound-ports-to-ignore,omitempty"`
}

// Kubernetes a K8s specific struct to hold config.
type Kubernetes struct {
	Kubeconfig string `json:"kubeconfig,omitempty"`
}

// CNIPluginConf is whatever JSON is passed via stdin.
type CNIPluginConf struct {
	types.NetConf

	LogLevel string `json:"log_level,omitempty"`

	Linkerd ProxyInit `json:"linkerd,omitempty"`

	Kubernetes Kubernetes `json:"kubernetes,omitempty"`
}

func newCNIPluginConf() *CNIPluginConf {
	return &CNIPluginConf{
		NetConf: types.NetConf{
			CNIVersion: constants.LinkerdCNIVersion,
			Name:       constants.LinkerdCNIName,
			Type:       constants.LinkerdCNIType,
		},
	}
}

func newMultusNetworkAttachDefinition(
	multusRef client.ObjectKey, config *CNIPluginConf) (*netattachv1.NetworkAttachmentDefinition, error) {
	var multusNetAttach = &netattachv1.NetworkAttachmentDefinition{
		TypeMeta: v1.TypeMeta{
			Kind:       constants.MultusNetworkAttachmentDefinitionResourceKind,
			APIVersion: constants.MultusNetworkAttachmentDefinitionAPIVersion,
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      multusRef.Name,
			Namespace: multusRef.Namespace,
		},
	}

	cfg, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	multusNetAttach.Spec.Config = string(cfg)

	return multusNetAttach, nil
}

// applyAttachDefinition configures provided CNIPluginConf with defined values from AttachDefinition.
func applyAttachDefinition(cfg *CNIPluginConf, ldAttach *cniv1alpha1.AttachDefinition) *CNIPluginConf {
	var ldCfg = ldAttach.Spec.Config

	if ldCfg.InboundPort != 0 {
		cfg.Linkerd.IncomingProxyPort = int(ldCfg.InboundPort)
	}

	if ldCfg.OutboundPort != 0 {
		cfg.Linkerd.OutgoingProxyPort = int(ldCfg.OutboundPort)
	}

	if ldCfg.ProxyUID != nil {
		cfg.Linkerd.ProxyUID = int(*ldCfg.ProxyUID)
	}

	if len(ldCfg.SkipInboundPorts) != 0 {
		for _, port := range ldCfg.SkipInboundPorts {
			if port.Port != 0 {
				cfg.Linkerd.InboundPortsToIgnore = append(cfg.Linkerd.InboundPortsToIgnore, strconv.Itoa(int(port.Port)))
			}

			if port.Range != "" {
				cfg.Linkerd.InboundPortsToIgnore = append(cfg.Linkerd.InboundPortsToIgnore, port.Range)
			}
		}
	}

	if len(ldCfg.SkipOutboundPorts) != 0 {
		for _, port := range ldCfg.SkipOutboundPorts {
			if port.Port != 0 {
				cfg.Linkerd.OutboundPortsToIgnore = append(cfg.Linkerd.OutboundPortsToIgnore, strconv.Itoa(int(port.Port)))
			}

			if port.Range != "" {
				cfg.Linkerd.OutboundPortsToIgnore = append(cfg.Linkerd.OutboundPortsToIgnore, port.Range)
			}
		}
	}

	return cfg
}
