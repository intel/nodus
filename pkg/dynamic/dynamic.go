package dynamic

import (
	"os"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
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

func (d *DynamicClient) GetResourceFromObject(gvk schema.GroupVersionKind) (dynamic.ResourceInterface, error) {

	gk := schema.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}

	// Get the available resources from the client
	groupResources, err := restmapper.GetAPIGroupResources(d.k8sclient.Discovery())
	if err != nil {
		return nil, err
	}

	// retrieve the required rest resource from the available mappings
	restMapper := restmapper.NewDiscoveryRESTMapper(groupResources)
	restMapping, err := restMapper.RESTMapping(gk, gvk.Version)
	if err != nil {
		return nil, err
	}

	// create a resource of that and return it
	resource := d.client.Resource(restMapping.Resource)
	if restMapping.Scope.Name() == apimeta.RESTScopeNameNamespace {
		// if namespaced, return the namespaced client
		return resource.Namespace(d.namespace), nil
	}
	return resource, nil
}

func (d *DynamicClient) getUnstructuredObjectFromFile(yamlPath string) (*unstructured.Unstructured, error) {
	// Conver the yaml into an unstructured object so that it can be consumed downstream
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
	// Get the group, version and kind of the new object
	gvk := object.GroupVersionKind()
	resourceInterface, err := d.GetResourceFromObject(gvk)
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
	// Get the group, version and kind of the new object
	gvk := object.GroupVersionKind()
	resourceInterface, err := d.GetResourceFromObject(gvk)
	if err != nil {
		return err
	}
	propagationPolicy := metav1.DeletePropagationForeground
	deleteOptions := &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}
	return resourceInterface.Delete(object.GetName(), deleteOptions)
}
