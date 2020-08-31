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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceLevelSpec defines the desired state of ServiceLevel
type ServiceLevelSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Address is the address of the Prometheus.
	PrometheusAddress string `json:"prometheusAddress"`
	// Name of Prometheus Operator Instance to integrate with
	PrometheusName string 	`json:"prometheusName"`
	ServiceLevelObjectives []SLO `json:"serviceLevelObjectives,omitempty"`
}

// ServiceLevelStatus defines the observed state of ServiceLevel
type ServiceLevelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	//Conditions status.Conditions `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ServiceLevel is the Schema for the servicelevels API
type ServiceLevel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceLevelSpec   `json:"spec,omitempty"`
	Status ServiceLevelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceLevelList contains a list of ServiceLevel
type ServiceLevelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceLevel `json:"items"`
}

// SLO defines the desired state of a Service Level Objective
type SLO struct {
	// Name of the SLO, must be made of [a-zA-z0-9] and '_'(underscore) characters.
	Name string `json:"name"`
	// Description is a description of the SLO.
	// +optional
	Description string `json:"description,omitempty"`
	// AvailabilityObjectivePercent is the percentage of availability target for the SLO.
	AvailabilityObjectivePercent string `json:"availabilityObjectivePercent"`
	// ServiceLevelIndicator is the SLI associated with the SLO.
	ServiceLevelIndicator SLI `json:"serviceLevelIndicator"`
	// Output is the output backedn of the SLO.
	Output Output `json:"output"`
}


// SLI is the SLI to get for the SLO.
type SLI struct {
	SLISource `json:",inline"`
}

// SLISource is where the SLI will get from.
type SLISource struct {
	// Prometheus is the prometheus SLI source.
	// +optional
	Prometheus *PrometheusSLISource `json:"prometheus,omitempty"`
}

// PrometheusSLISource is the source to get SLIs from a Prometheus backend.
type PrometheusSLISource struct {
	// TotalQuery is the query that gets the total that will be the base to get the unavailability
	// of the SLO based on the errorQuery (errorQuery / totalQuery).
	TotalQuery string `json:"totalQuery"`
	// ErrorQuery is the query that gets the total errors that then will be divided against the total.
	ErrorQuery string `json:"errorQuery"`
}

// Output is how the SLO will expose the generated SLO.
type Output struct {
	//Prometheus is the prometheus format for the SLO output.
	// +optional
	Prometheus *PrometheusOutputSource `json:"prometheus,omitempty"`
}

// PrometheusOutputSource  is the source of the output in prometheus format.
type PrometheusOutputSource struct {
	// Labels are the labels that will be set to the output metrics of this SLO.
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ServiceLevel{}, &ServiceLevelList{})
}
