/*
Copyright 2024.

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

package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	buildkitv1alpha1 "cops-buildkit/api/v1alpha1"
	"cops-buildkit/internal/buildkit"
)

// BuildkitReconciler reconciles a Buildkit object
type BuildkitReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=buildkit.thecops.dev,resources=buildkits,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=buildkit.thecops.dev,resources=buildkits/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=buildkit.thecops.dev,resources=buildkits/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Buildkit object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *BuildkitReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	instance := buildkitv1alpha1.Buildkit{}

	err := r.Get(context.TODO(), req.NamespacedName, &instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// object not found, could have been deleted after
			// reconcile request, hence don't requeue
			return ctrl.Result{}, nil
		}

		// error reading the object, requeue the request
		return ctrl.Result{}, err
	}

	// TODO
	// Create certs for demon and client, You can use mkcerts

	// Create a buildkit object
	bk := buildkit.Buildkit{
		Name:         req.Name,
		Namespace:    req.Namespace,
		Labels:       map[string]string{},
		NodeSelector: map[string]string{},
		Cloud:        instance.Spec.CloudProvider,
		Arch:         1,
		Image:        instance.Spec.Image,
		MaxReplica:   1,
		Resource:     corev1.ResourceRequirements{},
		Client:       r.Client,
	}

	if err := bk.CreateOrUpdateDeployment(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := bk.CreateOrUpdateService(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := bk.CreateOrUpdateSecret(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := bk.CreateOrUpdateConfig(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := bk.CreateOrUpdatePodDisruptionBudget(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := bk.CreateOrUpdateHorizontalPodAutoscalerionBudget(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *BuildkitReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&buildkitv1alpha1.Buildkit{}).
		Complete(r)
}
