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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"google.golang.org/protobuf/types/known/structpb"
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	networkv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	versionedclient "istio.io/client-go/pkg/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EndpointsReconciler EndPointReconciler reconciles a EndPoint object
type EndpointsReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	IstioClient *versionedclient.Clientset
}

//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;
//+kubebuilder:rbac:groups=networking.istio.io,resources=envoyfilters,verbs=get;list;create;delete;update;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the EndPoint object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *EndpointsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("endpoints", req.NamespacedName)

	var endpoints corev1.Endpoints
	name := req.NamespacedName.Name
	namespace := req.NamespacedName.Namespace
	if err := r.Client.Get(ctx, req.NamespacedName, &endpoints); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "get endpoints: ", name, namespace)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch endpoints")
		return ctrl.Result{}, err
	}

	if endpoints.Labels == nil {
		endpoints.Labels = make(map[string]string)
	}
	registryService := endpoints.Labels[registryServiceLabel] == "true"
	var endpointIps []interface{}
	envoyfilter, err := r.IstioClient.NetworkingV1alpha3().EnvoyFilters(namespace).Get(ctx, name,
		metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) || (envoyfilter.ObjectMeta.Name == "" && envoyfilter.ObjectMeta.
			Namespace == "") {
			envoyfilter = &networkv1alpha3.EnvoyFilter{
				ObjectMeta: v1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
		}
	}

	// 如果envoyfilter存在，并且没有注册中心服务的label或者endpoint被删除（svc不存在），删除该envoyfilter
	if !envoyfilter.CreationTimestamp.IsZero() && (!registryService || !endpoints.DeletionTimestamp.IsZero()) {
		log.Info("service not been registry service or has been deleted", namespace, name)
		_ = r.IstioClient.NetworkingV1alpha3().EnvoyFilters(namespace).Delete(ctx,
			name, metav1.DeleteOptions{})
	}

	if registryService {
		// update rds for xds
		// select all endpoints for this service
		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				endpointIps = append(endpointIps, address.IP)
			}
		}
		if len(endpointIps) == 0 {
			return ctrl.Result{}, nil
		}

		// 创建 envoyfilter 资源，
		values := map[string]interface{}{
			"domains": endpointIps,
		}
		valueStruct, err := structpb.NewStruct(values)
		if err != nil {
			log.Error(err, "build value has error")
		}
		envoyfilter.Spec = networkingv1alpha3.EnvoyFilter{
			ConfigPatches: []*networkingv1alpha3.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: networkingv1alpha3.EnvoyFilter_VIRTUAL_HOST,
					Match: &networkingv1alpha3.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: networkingv1alpha3.EnvoyFilter_SIDECAR_OUTBOUND,
						ObjectTypes: &networkingv1alpha3.EnvoyFilter_EnvoyConfigObjectMatch_RouteConfiguration{
							RouteConfiguration: &networkingv1alpha3.EnvoyFilter_RouteConfigurationMatch{
								Vhost: &networkingv1alpha3.EnvoyFilter_RouteConfigurationMatch_VirtualHostMatch{
									Name: name,
								},
							},
						},
					},
					Patch: &networkingv1alpha3.EnvoyFilter_Patch{
						Operation: networkingv1alpha3.EnvoyFilter_Patch_MERGE,
						Value:     valueStruct,
					},
				},
			},
		}
		if envoyfilter.CreationTimestamp.IsZero() {
			_, err := r.IstioClient.NetworkingV1alpha3().EnvoyFilters(namespace).Create(ctx,
				envoyfilter, metav1.CreateOptions{})
			if err != nil {
				log.Error(err, "create virtual service has error")
			}
		} else {
			_, err := r.IstioClient.NetworkingV1alpha3().EnvoyFilters(namespace).Update(ctx,
				envoyfilter, metav1.UpdateOptions{})
			if err != nil {
				log.Error(err, "update virtual service has error")
			}
		}

	}
	return ctrl.Result{}, nil
}

/***
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: custom-envoyfilter-httpbin
  namespace: default
spec:
  configPatches:
  - applyTo: VIRTUAL_HOST
    match:
      context: SIDECAR_OUTBOUND
      routeConfiguration:
        portNumber: 80
        vhost:
          name: "httpbin.default.svc.cluster.local:80"
    patch:
      operation: MERGE
      value:
        domains:
        - "10.0.2.119"
*/

// SetupWithManager sets up the controller with the Manager.
func (r *EndpointsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Endpoints{}).
		Complete(r)
}
