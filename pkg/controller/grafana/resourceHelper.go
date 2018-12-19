package grafana

import (
	yaml "github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	integreatlyv1alpha1 "github.com/integr8ly/grafana-operator/pkg/apis/integreatly/v1alpha1"

)

type ResourceHelper struct {
	templateHelper *GrafanaTemplateHelper
	cr             *integreatlyv1alpha1.Grafana
}

func newResourceHelper(cr *integreatlyv1alpha1.Grafana) *ResourceHelper {
	return &ResourceHelper{
		templateHelper: newTemplateHelper(cr),
		cr:             cr,
	}
}

func (r *ResourceHelper) createResource(template string) (runtime.Object, error) {
	tpl, err := r.templateHelper.loadTemplate(template)
	if err != nil {
		return nil, err
	}

	resource := unstructured.Unstructured{}
	err = yaml.Unmarshal(tpl, &resource)

	if err != nil {
		return nil, err
	}

	return &resource, nil
}