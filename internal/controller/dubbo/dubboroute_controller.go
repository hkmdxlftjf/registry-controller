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
	pkgctrl "github.com/hkmdxlftjf/registry-controller/internal"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	dubbov1 "github.com/hkmdxlftjf/registry-controller/api/dubbo/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DubboRouteReconciler reconciles a DubboRoute object
type DubboRouteReconciler struct {
	Log logr.Logger
	client.Client
	Scheme      *runtime.Scheme
	IstioClient *versionedclient.Clientset
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
	var dubboRoute = new(dubbov1.DubboRoute)
	var namespace = req.NamespacedName.Namespace
	var name = req.NamespacedName.Name
	if err := r.Client.Get(ctx, req.NamespacedName, dubboRoute); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "not found Dubbo Routes: ", "name", name,
				"namespace", namespace)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch endpoints")
		return ctrl.Result{}, err
	}
	deleted := dubboRoute.DeletionTimestamp != nil && !dubboRoute.DeletionTimestamp.IsZero()
	finalizer := sets.NewString(dubboRoute.GetFinalizers()...)
	if deleted {
		if !finalizer.Has(pkgctrl.DubboRouteFinalizer) {
			log.Info("dubbo route resource deleted", "name", name)
			return ctrl.Result{}, nil
		}
	} else if !finalizer.Has(pkgctrl.DubboRouteFinalizer) {
		log.Info("add finalizes to dubbo route", "name", name, "namespace", namespace)
		controllerutil.AddFinalizer(dubboRoute, pkgctrl.DubboRouteFinalizer)
		if err := r.Update(ctx, dubboRoute); err != nil {
			log.Error(err, "reconcile fail to add finalizer to dubbo",
				"controller", "DubboRouteReconciler", "name", name, "namespace", namespace)
		}
		return ctrl.Result{}, nil
	} else {
		// 1. 创建dr，创建envoyfilter
		var needUpdatePolicy bool

		var versions []string
		for _, config := range dubboRoute.Spec.RouteConfig {
			for _, route := range config.Routes {
				for _, r := range route.Route {
					versions = append(versions, r.Version)
				}
			}
		}
		if len(versions) != 0 {
			rule, err := r.IstioClient.NetworkingV1beta1().DestinationRules(namespace).Get(ctx, name, metav1.GetOptions{})
			if apierrors.IsNotFound(err) || (rule.ObjectMeta.Name == "" && rule.ObjectMeta.
				Namespace == "") {
				rule.Namespace = namespace
				rule.Name = name
			}
		}
	}

	if deleted {
		if requeue, err := r.handleDelete(ctx, log, &req); requeue || err != nil {
			//logger.Error(err, "handle delete failed",
			//	"controller", "MeshServiceReconciler",
			//	"name", req.Name, "namespace", req.Namespace)
			log.Error(err, "handle delete failed", "controller", "DubboRouteReconciler",
				"name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		controllerutil.RemoveFinalizer(dubboRoute, pkgctrl.DubboRouteFinalizer)
		if err := r.Update(ctx, dubboRoute); err != nil {
			if err := r.Update(ctx, dubboRoute); err != nil {
				if apierrors.IsNotFound(err) {
					log.Info("Did not remove finalizer: the object was previously deleted",
						"namespace", req.Namespace, "name", req.Name)
					return ctrl.Result{}, nil
				}
				log.Error(err, "reconcile fail to delete dubbo route finalizer",
					"controller", "DubboRouteReconciler", "name", req.Name, "namespace", req.Namespace)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// handleDelete handle dr and envoyfilter after dubborouter delete
func (r *DubboRouteReconciler) handleDelete(ctx context.Context, log logr.Logger, req *ctrl.Request) (requeue bool, err error) {
	if err := r.IstioClient.NetworkingV1beta1().DestinationRules(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "delete destination rule failed, not found ", "name", req.Name, "namespace", req.Namespace)
		}
		requeue = true
		log.Error(err, "delete destination rule failed")
		return
	}
	if err := r.IstioClient.NetworkingV1alpha3().EnvoyFilters(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "delete envoyfilter failed, not found ", "name", req.Name, "namespace", req.Namespace)
		}
		requeue = true
		log.Error(err, "delete envoyfilter rule failed")
		return false, err
	}

	return requeue, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DubboRouteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dubbov1.DubboRoute{}).
		Complete(r)
}
