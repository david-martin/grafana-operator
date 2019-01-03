package e2e

import (
	goctx "context"
	"testing"
	"time"

	"github.com/integr8ly/grafana-operator/pkg/apis"
	integreatlyv1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
)

var (
	retryInterval        = time.Second * 5
	timeout              = time.Second * 60
	cleanupRetryInterval = time.Second * 1
	cleanupTimeout       = time.Second * 5
)

func TestGrafana(t *testing.T) {
	// register the operator's scheme with the framework's dynamic client
	grafanaList := &integreatlyv1alpha1.GrafanaList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Grafana",
			APIVersion: "org.integreatly/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, grafanaList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	// create a TestCtx for the current test and defer its cleanup function
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	// initialize the test's kubernetes resources
	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: cleanupTimeout, RetryInterval: cleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}

	// get namespace
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for grafana-operator to be ready
	err = e2eutil.WaitForOperatorDeployment(t, f.KubeClient, namespace, "grafana-operator", 1, time.Second*5, time.Second*30)
	if err != nil {
		t.Fatal(err)
	}

	// create grafana custom resource
	exampleGrafana := &integreatlyv1alpha1.Grafana{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Grafana",
			APIVersion: "org.integreatly/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-grafana",
			Namespace: namespace,
		},
		Spec: integreatlyv1alpha1.GrafanaSpec{
			PrometheusUrl: "http://localhost:9090",
		},
	}
	err = f.Client.Create(goctx.TODO(), exampleGrafana, &framework.CleanupOptions{TestContext: ctx, Timeout: time.Second * 5, RetryInterval: time.Second * 1})
	if err != nil {
		t.Fatalf("failed to create grafana: %v", err)
	}

	// wait for example-grafana to reach 1 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "grafana-deployment", 1, time.Second*5, time.Second*30)
	if err != nil {
		t.Fatalf("failed waiting for deployment: %v", err)
	}
}
