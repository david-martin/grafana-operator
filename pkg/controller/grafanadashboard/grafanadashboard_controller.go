package grafanadashboard

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/types"
	"fmt"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = logf.Log.WithName("controller_grafanadashboard")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new GrafanaDashboard Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGrafanaDashboard{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("grafanadashboard-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource GrafanaDashboard
	err = c.Watch(&source.Kind{Type: &integreatlyv1alpha1.GrafanaDashboard{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner GrafanaDashboard
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &integreatlyv1alpha1.GrafanaDashboard{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileGrafanaDashboard{}

// ReconcileGrafanaDashboard reconciles a GrafanaDashboard object
type ReconcileGrafanaDashboard struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a GrafanaDashboard object and makes changes based on the state read
// and what is in the GrafanaDashboard.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGrafanaDashboard) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GrafanaDashboard")

	// Fetch the GrafanaDashboard instance
	instance := &integreatlyv1alpha1.GrafanaDashboard{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.Created == false {
		err = r.UpdateGrafanaConfigMap(instance)
		if err != nil {
			return reconcile.Result{Requeue: true}, err
		}
		r.UpdatePhase(instance)
	} else {
		log.Info(fmt.Sprintf("%s already created", instance.Spec.Name))
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileGrafanaDashboard) UpdatePhase(cr *integreatlyv1alpha1.GrafanaDashboard) error {
	cr.Status.Created = true
	return r.client.Update(context.TODO(), cr)
}

func (r *ReconcileGrafanaDashboard) UpdateGrafanaConfigMap(cr *integreatlyv1alpha1.GrafanaDashboard) error {
	// Get the application monitoring namespace
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Error getting operator namespace")
		return err
	}

	selector := types.NamespacedName{
		Namespace: namespace,
		Name: "grafana-dashboards",
	}

	log.Info(fmt.Sprintf("Looking for config map in %s", selector))

	resource := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "grafana-dashboards",
			Namespace: namespace,
		},
	}

	err = r.client.Get(context.TODO(), selector, &resource)
	if err != nil {
		if errors.IsNotFound(err) {
			r.client.Create(context.TODO(), &resource)
			log.Info("New grafana dashboards config map created")
		} else {
			log.Error(err, "Error looking up grafana dashboards config map")
			return err
		}
	}

	if resource.Data == nil {
		resource.Data = make(map[string]string)
	}

	// Set the CR as the owner of this resource so that when
	// the CR is deleted this resource also gets removed
	err = controllerutil.SetControllerReference(cr, &resource, r.scheme)
	if err != nil {
		return fmt.Errorf("Error setting the custom resource as owner: %s", err)
		return err
	}

	resource.Data[cr.Spec.Name] = cr.Spec.Json
	r.client.Update(context.TODO(), &resource)
	log.Info("New dashboard added to config map")
	return nil
}
