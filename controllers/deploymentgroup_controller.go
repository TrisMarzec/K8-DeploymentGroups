package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "k8-deploymentgroups/api/v1alpha1"
)

// DeploymentGroupReconciler reconciles a DeploymentGroup object
type DeploymentGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8-deploymentgroups.io,resources=deploymentgroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8-deploymentgroups.io,resources=deploymentgroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8-deploymentgroups.io,resources=deploymentgroups/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

func (r *DeploymentGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the DeploymentGroup instance
	var deploymentGroup v1alpha1.DeploymentGroup
	if err := r.Get(ctx, req.NamespacedName, &deploymentGroup); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 0. Validate Dependencies (Cycle Detection)
	if err := r.checkForCycles(deploymentGroup.Spec.Items); err != nil {
		log.Error(err, "Cycle detected in deployment dependencies")
		// Update status to CycleDetected
		// We could add a Condition here.
		return ctrl.Result{}, nil // Stop processing
	}

	// 1. Build a map of items and their current state
	// Map: Name -> DeploymentItem
	itemsMap := make(map[string]v1alpha1.DeploymentItem)
	for _, item := range deploymentGroup.Spec.Items {
		itemsMap[item.Name] = item
	}

	// 2. Fetch all deployments and determine their readiness
	// Map: Name -> isReady
	readyStatus := make(map[string]bool)
	// Map: Name -> current deployment object
	deployments := make(map[string]*appsv1.Deployment)

	for _, item := range deploymentGroup.Spec.Items {
		var dep appsv1.Deployment
		err := r.Get(ctx, types.NamespacedName{Name: item.Name, Namespace: deploymentGroup.Namespace}, &dep)
		if err != nil {
			if errors.IsNotFound(err) {
				log.Info("Deployment not found", "name", item.Name)
				// We consider it not ready. We can't act on it yet.
				readyStatus[item.Name] = false
				continue
			}
			return ctrl.Result{}, err
		}
		deployments[item.Name] = &dep

		// Check readiness
		target := int32(1)
		if item.TargetReplicas != nil {
			target = *item.TargetReplicas
		}

		isReady := dep.Status.AvailableReplicas >= target && dep.Status.ObservedGeneration == dep.Generation
		readyStatus[item.Name] = isReady
	}

	// 3. Enact state
	var newReadyItems []string

	for _, item := range deploymentGroup.Spec.Items {
		// Check dependencies
		allDepsReady := true
		for _, depName := range item.DependsOn {
			if !readyStatus[depName] {
				allDepsReady = false
				break
			}
		}

		currentDep, exists := deployments[item.Name]
		if !exists {
			continue // Can't update what doesn't exist
		}

		desiredReplicas := int32(0)
		if allDepsReady {
			if item.TargetReplicas != nil {
				desiredReplicas = *item.TargetReplicas
			} else {
				desiredReplicas = 1
			}
			// If it's ready, add to status list
			if readyStatus[item.Name] {
				newReadyItems = append(newReadyItems, item.Name)
			}
		}

		if currentDep.Spec.Replicas == nil || *currentDep.Spec.Replicas != desiredReplicas {
			log.Info("Updating deployment replicas", "name", item.Name, "desired", desiredReplicas)
			currentDep.Spec.Replicas = &desiredReplicas
			if err := r.Update(ctx, currentDep); err != nil {
				return ctrl.Result{}, err
			}
			// We updated the deployment, next reconcile will pick up the new generation OR we wait.
		}
	}

	// 4. Update Status
	deploymentGroup.Status.ReadyItems = newReadyItems
	// TODO: Set Conditions
	if err := r.Status().Update(ctx, &deploymentGroup); err != nil {
		return ctrl.Result{}, err
	}

	// If we made updates or are waiting for things to become ready, we should reconcile again?
	// The standard watch on Deployments should trigger us if we own them or if we set up a watcher.
	// We will need to setup the specific Controller Builder to watch Deployments.

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *DeploymentGroupReconciler) checkForCycles(items []v1alpha1.DeploymentItem) error {
	// Build Adjacency List
	adj := make(map[string][]string)
	for _, item := range items {
		adj[item.Name] = item.DependsOn
	}

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}
		recStack[node] = false
		return false
	}

	for _, item := range items {
		if !visited[item.Name] {
			if dfs(item.Name) {
				return fmt.Errorf("cycle detected involving %s", item.Name)
			}
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.DeploymentGroup{}).
		// TODO: specific watch for deployments mentioned in groups?
		// For simplicity, we can watch all Deployments, or rely on polling (RequeueAfter) + ownerRefs if we added them.
		// Since we don't own the Deployments (they might exist independently), a general Watch + Filter is better.
		// But for MVP, RequeueAfter 10s is safe fallback.
		Complete(r)
}
