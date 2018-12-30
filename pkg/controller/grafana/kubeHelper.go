package grafana

import (
	gr "github.com/integr8ly/grafana-operator/pkg/client/versioned"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type KubeHelper interface {
	listNamespaces()
}

type KubeHelperImpl struct {
	k8client *kubernetes.Clientset
	grclient *gr.Clientset
}

func newKubeHelper() *KubeHelperImpl {
	config := config.GetConfigOrDie()

	k8client := kubernetes.NewForConfigOrDie(config)
	grclient := gr.NewForConfigOrDie(config)

	helper := new(KubeHelperImpl)
	helper.k8client = k8client
	helper.grclient = grclient
	return helper
}

func (h KubeHelperImpl) getMonitoringNamespaces() ([]v1.Namespace, error) {
	selector := metav1.ListOptions{
		LabelSelector: "monitoring=enabled",
	}

	namespaces, err := h.k8client.CoreV1().Namespaces().List(selector)
	if err != nil {
		return nil, err
	}

	return namespaces.Items, nil
}

func (h KubeHelperImpl) getNamespaceDashboards(namespaceName string) error {
	selector := metav1.ListOptions{}
	dashboards, err := h.grclient.IntegreatlyV1alpha1().GrafanaDashboards(namespaceName).List(selector)

	if err != nil {
		return err
	}

	log.Info("Dashboards: %s", dashboards)

	return nil
}
