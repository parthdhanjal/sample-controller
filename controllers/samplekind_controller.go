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

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/parthdhanjal/sample-controller/api/v1alpha1"
	handler "github.com/parthdhanjal/sample-controller/handler"
)

var log = ctrl.Log.WithName("controllers").WithName("samplekind")

// SampleKindReconciler reconciles a SampleKind object
type SampleKindReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.sample.com,resources=samplekinds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.sample.com,resources=samplekinds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.sample.com,resources=samplekinds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SampleKind object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *SampleKindReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	fmt.Print("----------------Reconciling----------------")
	instance := &cachev1alpha1.SampleKind{}
	err := r.Get(ctx, req.Namespace, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Requested Resource not found. Resource deletion confirmed")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Faiiled to get resource. Resource not available")
		return ctrl.Result{}, nil
	}

	return handler.Handle(ctx, instance)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SampleKindReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.SampleKind{}).
		Owns(&core.Pod{}).
		Complete(r)
}
