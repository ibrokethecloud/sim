package objects

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// ingressCleanup object specific cleanup
func ingressCleanup(obj *unstructured.Unstructured) error {
	obj.SetAPIVersion("networking.k8s.io/v1")
	return nil
}

// jobCleanup performs job specific cleanup
func jobCleanup(obj *unstructured.Unstructured) error {
	labels := obj.GetLabels()
	copyKeyValue(labels, "controller-uid", simLabelPrefix+"controller-uid")
	obj.SetLabels(labels)

	unstructured.RemoveNestedField(obj.Object, "spec", "template", "metadata", "labels")
	unstructured.RemoveNestedField(obj.Object, "spec", "selector")
	return nil
}

// copyKeyValue is a helper function to copy a kv and create a new kv in the map with new key but same value. Needed to help maintain resource versions when needed
func copyKeyValue(obj map[string]string, key string, newKey string) {
	v, ok := obj[key]
	if ok {
		obj[newKey] = v
		delete(obj, key)
	}
}

// cleans the apiService to point to no-where
func apiServiceCleanup(obj *unstructured.Unstructured) error {
	unstructured.RemoveNestedField(obj.Object, "spec", "service")
	unstructured.RemoveNestedField(obj.Object, "spec", "caBundle")
	unstructured.RemoveNestedField(obj.Object, "spec", "insecureSkipTLSVerify")
	return nil
}
