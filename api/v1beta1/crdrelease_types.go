/*


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

package v1beta1

import (
	"cloudnativeapp/clm/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CRDReleaseSpec defines the desired state of CRDRelease
type CRDReleaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The version of CRDRelease to be intalled, Unique combine with release name
	Version string `json:"version,omitempty"`
	// Dependencies to install this CRDRelease
	Dependencies []internal.Dependency `json:"dependencies,omitempty"`
	// CRDRelease consist of multi modules, every module implements part of functions of release
	Modules []internal.Module `json:"modules,omitempty"`
}

// CRDReleaseStatus defines the observed state of CRDRelease
type CRDReleaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions   []internal.CRDReleaseCondition `json:"conditions,omitempty"`
	Dependencies []internal.DependencyStatus    `json:"dependencies,omitempty"`
	Modules      []internal.ModuleStatus        `json:"modules,omitempty"`
	// CurrentVersion changed after install release successully.
	CurrentVersion string                   `json:"currentVersion,omitempty"`
	Phase          internal.CRDReleasePhase `json:"phase,omitempty"`
	Reason         string                   `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// CRDRelease is the Schema for the crdreleases API
type CRDRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CRDReleaseSpec   `json:"spec,omitempty"`
	Status CRDReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CRDReleaseList contains a list of CRDRelease
type CRDReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CRDRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CRDRelease{}, &CRDReleaseList{})
}
