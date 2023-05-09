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

package dubbo

import (
	"context"
	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	dubbov1 "registry-controller/api/dubbo/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DubboRouteReconciler reconciles a DubboRoute object
type DubboRouteReconciler struct {
	Log logr.Logger
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=dubbo.org.bigfemonkey,resources=dubboroutes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=dubbo.org.bigfemonkey,resources=dubboroutes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=dubbo.org.bigfemonkey,resources=dubboroutes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DubboRoute object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *DubboRouteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("dubboroutes", req.NamespacedName)
	log.Info("dubbo route on create", "name", req.Name, "namespace", req.Namespace)
	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DubboRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dubbov1.DubboRoute{}).
		Complete(r)
}
