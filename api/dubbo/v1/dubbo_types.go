/*
Copyright 2023.

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
	"istio.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DubboSpec defines the desired state of Dubbo
type DubboSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	RouteConfig []RouteConfig `json:"route_config,omitempty"`
}

type Routes struct {
	Method map[string]v1beta1.StringMatch `json:"method,omitempty"`
	Route  Route                          `json:"route"`
}

type Route struct {
	Cluster string `json:"cluster,omitempty"`
}

type RouteConfig struct {
	Interface string   `json:"interface,omitempty"`
	Name      string   `json:"name,omitempty"`
	Routes    []Routes `json:"routes"`
}

// DubboStatus defines the observed state of Dubbo
type DubboStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status []Status `json:"status,omitempty"`
}

type Status struct {
	Interface string `json:"interface,omitempty"`
	Method    string `json:"method"`
}

type Destination struct {
	Subset string `json:"subset,omitempty"`
	Weight string `json:"weight"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:path=mykinds,scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Spec",type="string",JSONPath=".spec.fieldName"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase"

// Dubbo is the Schema for the dubboes API
type Dubbo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DubboSpec   `json:"spec,omitempty"`
	Status DubboStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DubboList contains a list of Dubbo
type DubboList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dubbo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Dubbo{}, &DubboList{})
}
