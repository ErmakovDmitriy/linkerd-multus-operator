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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=65535

// Port used to define network port annotations at one place.
type Port uint16

// Ports defines ports of port ranges for Linkerd Proxy.
type Ports struct {
	Port Port `json:"port,omitempty" yaml:"port,omitempty"`
	// +kubebuilder:validation:Pattern="^((6553[0-5])|(655[0-2][0-9])|(65[0-4][0-9]{2})|(6[0-4][0-9]{3})|([1-5][0-9]{4})|([0-5]{0,5})|([0-9]{1,4}))-((6553[0-5])|(655[0-2][0-9])|(65[0-4][0-9]{2})|(6[0-4][0-9]{3})|([1-5][0-9]{4})|([0-5]{0,5})|([0-9]{1,4}))$"
	Range string `json:"range,omitempty" yaml:"range,omitempty"`
}

// // ContainerResourcesSet Linkerd Proxy container resources set.
// type ContainerResourcesSet struct {
// 	CPU    *resource.Quantity `json:"cpu,omitempty" yaml:"cpu,omitempty"`
// 	Memory *resource.Quantity `json:"memory,omitempty" yaml:"memory,omitempty"`
// }

// // ContainerResources  Linkerd Proxy container resource limits and requests.
// type ContainerResources struct {
// 	Requests *ContainerResourcesSet `json:"requests,omitempty" yaml:"requests,omitempty"`
// 	Limits   *ContainerResourcesSet `json:"limits,omitempty" yaml:"limits,omitempty"`
// }

// // ContainerImage defines container image for Proxy, Debug etc. containers.
// type ContainerImage struct {
// 	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
// 	Version    string `json:"version,omitempty" yaml:"version,omitempty"`
// 	PullPolicy string `json:"pullPolicy,omitempty" yaml:"pullPolicy,omitempty"`
// }

type ProxyConfig struct {
	// // config.linkerd.io/proxy-await.
	// ProxyAwait *bool `json:"proxyAwait,omitempty" yaml:"proxyAwait,omitempty"`

	// // config.linkerd.io/admin-port.
	// AdminPort Port `json:"adminPort,omitempty" yaml:"adminPort,omitempty"`
	// // config.linkerd.io/control-port.
	// ControlPort Port `json:"controlPort,omitempty" yaml:"controlPort,omitempty"`
	// config.linkerd.io/inbound-port.
	InboundPort Port `json:"inboundPort,omitempty" yaml:"inboundPort,omitempty"`
	// config.linkerd.io/outbound-port.
	OutboundPort Port `json:"outboundPort,omitempty" yaml:"outboundPort,omitempty"`

	// // config.linkerd.io/opaque-ports.
	// OpaquePorts []Ports `json:"opaquePorts,omitempty" yaml:"opaquePorts,omitempty"`
	// config.linkerd.io/skip-inbound-ports.
	SkipInboundPorts []Ports `json:"skipInboundPorts,omitempty" yaml:"skipInboundPorts,omitempty"`
	// config.linkerd.io/skip-outbound-ports.
	SkipOutboundPorts []Ports `json:"skipOutboundPorts,omitempty" yaml:"skipOutboundPorts,omitempty"`

	// // config.alpha.linkerd.io/proxy-wait-before-exit-seconds.
	// WaitBeforeExitSec uint32 `json:"waitBeforeExitSec,omitempty" yaml:"waitBeforeExitSec,omitempty"`
	// // config.linkerd.io/proxy-outbound-connect-timeout.
	// OutboundConnectTimeoutSec uint32 `json:"outboundConnectTimeoutSec,omitempty" yaml:"outboundConnectTimeoutSec,omitempty"`
	// // config.linkerd.io/close-wait-timeout.
	// CloseWaitTimeoutSec uint32 `json:"closeWaitTimeoutSec,omitempty" yaml:"closeWaitTimeoutSec,omitempty"`

	// // config.linkerd.io/enable-debug-sidecar.
	// EnableDebugSidecar *bool `json:"enableDebugSidecar,omitempty" yaml:"enableDebugSidecar,omitempty"`
	// // config.linkerd.io/debug-image (version, image, pull policy).
	// DebugImage ContainerImage `json:"debugImage,omitempty" yaml:"debugImage,omitempty"`
	// // config.linkerd.io/image-pull-policy, name, version.
	// ProxyImage ContainerImage `json:"proxyImage,omitempty" yaml:"proxyImage,omitempty"`
	// // config.linkerd.io/init-image, version, name,
	// // if the init image is used.
	// InitImage ContainerImage `json:"initImage,omitempty" yaml:"initImage,omitempty"`

	// // config.linkerd.io/disable-identity.
	// DisableIdentity *bool `json:"disableIdentity,omitempty" yaml:"disableIdentity,omitempty"`
	// // config.linkerd.io/enable-external-profiles.
	// EnableExternalProfiles *bool `json:"enableExternalProfiles,omitempty" yaml:"enableExternalProfiles,omitempty"`

	// // Proxy CPU, memory requests and limits, i.e.:
	// // config.linkerd.io/proxy-cpu-limit
	// // config.linkerd.io/proxy-cpu-request
	// // config.linkerd.io/proxy-memory-limit
	// // config.linkerd.io/proxy-memory-request
	// Resources ContainerResources `json:"resources,omitempty" yaml:"resources,omitempty"`

	// // +kubebuilder:validation:Enum=plain;json
	// // config.linkerd.io/proxy-log-format.
	// LogFormat string `json:"logFormat,omitempty" yaml:"logFormat,omitempty"`
	// config.linkerd.io/proxy-log-level.
	LogLevel string `json:"logLevel,omitempty" yaml:"logLevel,omitempty"`

	// +kubebuilder:validation:Required
	// config.linkerd.io/proxy-version.
	ProxyUID *uint32 `json:"proxyUID,omitempty" yaml:"proxyUID,omitempty"`
}

// AttachDefinitionSpec defines the desired state of AttachDefinition
type AttachDefinitionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default=true

	// CreateMultusNetworkAttachmentDefinition if set, then the controller
	// will generate a k8s.cni.cncf.io/v1 NetworkAttachmentDefinition
	// to trigger Multus to call Linkerd CNI on a Pod start.
	CreateMultusNetworkAttachmentDefinition bool `json:"createMultusNetworkAttachmentDefinition,omitempty" yaml:"createMultusNetworkAttachmentDefinition,omitempty"`

	// ProxyConfig configures Proxy via annotations.
	// Further below in comments are the annotations which will be added to a Pod.
	// https://linkerd.io/2.11/reference/proxy-configuration/ .
	Config ProxyConfig `json:"proxyConfig,omitempty" yaml:"proxyConfig,omitempty"`
}

// AttachDefinitionStatus defines the observed state of AttachDefinition
type AttachDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AttachDefinition is the Schema for the attachdefinitions API
type AttachDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AttachDefinitionSpec   `json:"spec,omitempty"`
	Status AttachDefinitionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AttachDefinitionList contains a list of AttachDefinition
type AttachDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AttachDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AttachDefinition{}, &AttachDefinitionList{})
}
