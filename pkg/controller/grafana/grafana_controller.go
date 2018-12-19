package grafana

import (
	"context"

	integreatlyv1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

)

var log = logf.Log.WithName("controller_grafana")

const (
	PhaseConfigFiles = iota
	PhaseInstallGrafana
	PhaseDone
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Grafana Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGrafana{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("grafana-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Grafana
	err = c.Watch(&source.Kind{Type: &integreatlyv1alpha1.Grafana{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Grafana
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &integreatlyv1alpha1.Grafana{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileGrafana{}

// ReconcileGrafana reconciles a Grafana object
type ReconcileGrafana struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Grafana object and makes changes based on the state read
// and what is in the Grafana.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGrafana) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Grafana")

	// Fetch the Grafana instance
	instance := &integreatlyv1alpha1.Grafana{}
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

	instanceCopy := instance.DeepCopy()

	switch instanceCopy.Status.Phase {
	case PhaseConfigFiles:
		return r.CreateConfigFiles(instanceCopy)
	case PhaseInstallGrafana:
		return r.InstallGrafana(instanceCopy)
	case PhaseDone:
		log.Info("Grafana installation complete")
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileGrafana) CreateConfigFiles(cr *integreatlyv1alpha1.Grafana) (reconcile.Result, error) {
	log.Info("Phase: Create Config Files")

	for _, resourceName := range []string{GrafanaServiceAccountName,GrafanaConfigMapName, GrafanaDashboardsConfigMapName, GrafanaProvidersConfigMapName, GrafanaDatasourcesConfigMapName} {
		if err := r.CreateResource(cr, resourceName); err != nil {
			log.Info("Error in CreateConfigFiles, resourceName=%s : err=%s", resourceName, err)
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{Requeue: true}, r.UpdatePhase(cr, PhaseInstallGrafana)
}

func (r *ReconcileGrafana) InstallGrafana(cr *integreatlyv1alpha1.Grafana) (reconcile.Result, error) {
	log.Info("Phase: Install Grafana")

	for _, resourceName := range []string{GrafanaDeploymentName} {
		if err := r.CreateResource(cr, resourceName); err != nil {
			log.Info("Error in InstallGrafana, resourceName=%s : err=%s", resourceName, err)
			// Requeue so it can be attempted again
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{Requeue: true}, r.UpdatePhase(cr, PhaseInstallGrafana)
}


func (r *ReconcileGrafana) UpdatePhase(cr *integreatlyv1alpha1.Grafana, phase int) error {
	cr.Status.Phase = phase
	return r.client.Update(context.TODO(), cr)
}

// Creates a generic kubernetes resource from a templates
func (r *ReconcileGrafana) CreateResource(cr *integreatlyv1alpha1.Grafana, resourceName string) error {
	resourceHelper := newResourceHelper(cr)
	resource, err := resourceHelper.createResource(resourceName)

	if err != nil {
		return fmt.Errorf("Error parsing templates: %s", err)
	}

	// Try to find the resource, it may already exist
	selector := types.NamespacedName{
		Namespace: cr.Namespace,
		Name:      resourceName,
	}
	err = r.client.Get(context.TODO(), selector, resource)

	// The resource exists, do nothing
	if err == nil {
		return nil
	}

	// Resource does not exist or something went wrong
	if errors.IsNotFound(err) {
		log.Info("Resource '%s' is missing. Creating it.", resourceName)
	} else {
		return fmt.Errorf("Error reading resource '%s': %s", resourceName, err)
	}

	// Set the CR as the owner of this resource so that when
	// the CR is deleted this resource also gets removed
	err = controllerutil.SetControllerReference(cr, resource.(v1.Object), r.scheme)
	if err != nil {
		return fmt.Errorf("Error setting the custom resource as owner: %s", err)
	}

	err = r.client.Create(context.TODO(), resource)
	if err != nil {
		return fmt.Errorf("Error creating resource: %s", err)
	}
	return nil
}