package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentGroupSpec defines the desired state of DeploymentGroup
type DeploymentGroupSpec struct {
	// Items defines the list of deployments and their dependencies
	Items []DeploymentItem `json:"items"`
}

type DeploymentItem struct {
	// Name of the Deployment (must be in the same namespace as the DeploymentGroup)
	Name string `json:"name"`
	// TargetReplicas specifies the desired replicas when the dependency is met.
	// If not set, it could default to 1 or look for an annotation on the Deployment.
	TargetReplicas *int32 `json:"targetReplicas,omitempty"`
	// DependsOn is a list of names of other DeploymentItems in this group that must be Ready first
	DependsOn []string `json:"dependsOn,omitempty"`
}

// DeploymentGroupStatus defines the observed state of DeploymentGroup
type DeploymentGroupStatus struct {
	// Conditions tracks the overall status
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ReadyItems lists the names of deployments that are currently considered Ready
	ReadyItems []string `json:"readyItems,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DeploymentGroup is the Schema for the deploymentgroups API
type DeploymentGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeploymentGroupSpec   `json:"spec,omitempty"`
	Status DeploymentGroupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DeploymentGroupList contains a list of DeploymentGroup
type DeploymentGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentGroup{}, &DeploymentGroupList{})
}
