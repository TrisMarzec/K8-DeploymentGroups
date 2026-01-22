package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeploymentGroupSpec defines the desired state of a DeploymentGroup, including the list of deployments and their dependencies.
type DeploymentGroupSpec struct {
	// Items is the list of deployments to be managed, including their dependencies and configuration overrides.
	Items []DeploymentItem `json:"items"`
}

type DeploymentItem struct {
	// Name corresponds to the name of the Kubernetes Deployment resource.
	// The Deployment must exist in the same namespace as the DeploymentGroup.
	Name string `json:"name"`
	// TargetReplicas specifies the number of replicas the deployment should have when all its dependencies are met.
	// If not specified, it defaults to 1.
	TargetReplicas *int32 `json:"targetReplicas,omitempty"`
	// DependsOn is a list of other DeploymentItem names within this group that must be in a Ready state before this deployment can be scaled up.
	DependsOn []string `json:"dependsOn,omitempty"`
	// PriorityClassName specifies the PriorityClass to be assigned to the Deployment's Pods.
	// If specified, the controller will update the Deployment's .spec.template.spec.priorityClassName to match this value.
	PriorityClassName *string `json:"priorityClassName,omitempty"`
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
