/*
Copyright 2021.

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
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=routes,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Service object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// check if service is of type *-driver-svc
	isASparkService := strings.Contains(req.Name, "driver-svc")
	// if it is not a service kind we are monitoring, then exit
	if !isASparkService {
		return ctrl.Result{}, nil
	}

	// 1. fetch service from kube API - every time a svc is created, updated or deleted
	svc := &corev1.Service{}
	if err := r.Get(ctx, req.NamespacedName, svc); err != nil {
		if apierrors.IsNotFound(err) {
			// the service was deleted, check if a route exist with the same name
			route := &routev1.Route{}
			if err := r.Get(ctx, req.NamespacedName, route); err != nil {
				if apierrors.IsNotFound(err) {
					// no need to do anything, return successfully
					return ctrl.Result{}, nil
				}
			}
			// a route with the same name was found, delete it
			if err := r.Delete(ctx, route); err != nil {
				log.Log.Error(err, "Impossible to delete the route for the service")
				return ctrl.Result{}, err
			}
		}
		// Error reading the object - requeue the request.
		log.Log.Error(err, "unable to fetch Service")
		return ctrl.Result{}, err
	}

	// 2. a spark-ui service was found

	// check if a route already exists for the service
	route := &routev1.Route{}
	if err := r.Get(ctx, req.NamespacedName, route); err != nil {
		if apierrors.IsNotFound(err) {
			// the route was not found, so we got to create it
			newRoute := r.getRouteForSparkUI(svc)
			if err := r.Create(ctx, newRoute); err != nil {
				log.Log.Error(err, "Impossible to create a route for the service")
				return ctrl.Result{}, err
			}
			// the route was create successfully, return nil
			return ctrl.Result{}, nil
		}
		// any other error is ignored
	}

	// a route already exist
	// if an error was triggered, attempt retrieving it as ingress

	// any other service type is ignored

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) getRouteForSparkUI(svc *corev1.Service) *routev1.Route {

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Labels:    svc.Labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: svc.Name,
			},
			Port: &routev1.RoutePort{
				//TargetPort: intstr.FromString(svc.Name),
				TargetPort: intstr.FromString("spark-ui"),
			},
			/*
				TLS: &routev1.TLSConfig{
					Termination: routev1.TLSTerminationEdge,
				},
			*/
		},
	}

	// Set MobileSecurityService mss as the owner and controller
	controllerutil.SetControllerReference(svc, route, r.Scheme)
	return route
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r)
}
