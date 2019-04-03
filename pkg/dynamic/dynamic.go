package dynamic

import (
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	dynamic "k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

type DynamicClient struct {
	client    dynamic.Interface
	k8sclient kubernetes.Interface
	namespace string
}

func NewDynamicClient(dynamicClient dynamic.Interface, k8sClient kubernetes.Interface, namespace string) *DynamicClient {
	return &DynamicClient{
		client:    dynamicClient,
		k8sclient: k8sClient,
		namespace: namespace,
	}
}

func (d *DynamicClient) getResourceFromObject(object *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	gk := schema.GroupKind{
		Group: object.GroupVersionKind().Group,
		Kind:  object.GroupVersionKind().Kind,
	}

	groupResources, err := restmapper.GetAPIGroupResources(d.k8sclient.Discovery())
	if err != nil {
		return nil, err
	}

	restMapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	restMapping, err := restMapper.RESTMapping(gk, object.GroupVersionKind().Version)
	if err != nil {
		return nil, err
	}

	return d.client.Resource(restMapping.Resource).Namespace(d.namespace), nil
}

func (d *DynamicClient) getUnstructuredObjectFromFile(yamlPath string) (*unstructured.Unstructured, error) {
	reader, err := os.Open(yamlPath)
	if err != nil {
		return nil, err
	}

	object := &unstructured.Unstructured{}

	decoder := k8syaml.NewYAMLToJSONDecoder(reader)
	err = decoder.Decode(object)
	if err != nil {
		return nil, err
	}

	return object, nil
}

func (d *DynamicClient) Create(yamlPath string) error {

	object, err := d.getUnstructuredObjectFromFile(yamlPath)
	if err != nil {
		return err
	}
	resourceInterface, err := d.getResourceFromObject(object)
	if err != nil {
		return err
	}
	_, err = resourceInterface.Create(object, metav1.CreateOptions{})
	return err
}

func (d *DynamicClient) Delete(yamlPath string) error {
	object, err := d.getUnstructuredObjectFromFile(yamlPath)
	if err != nil {
		return err
	}

	resourceInterface, err := d.getResourceFromObject(object)
	if err != nil {
		return err
	}
	propagationPolicy := metav1.DeletePropagationBackground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}
	return resourceInterface.Delete(object.GetName(), deleteOptions)
}
