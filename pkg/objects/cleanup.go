package objects

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

// nodeCleanup patches the address in node to localhost and saves existing address
// into annotations
func nodeCleanup(obj *unstructured.Unstructured) error {
	if obj.GroupVersionKind().Group != "" || obj.GroupVersionKind().Version != "v1" {
		// kind Node may be present in other GVK
		// this ensures we patch nothing else
		return nil
	}

	status, ok, err := unstructured.NestedFieldCopy(obj.Object, "status")
	if err != nil {
		return err
	}
	if !ok {
		return nil //nothing to do
	}

	statusMap, ok := status.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unable to assert status as a map[string]interface{}")
	}
	addresses, ok := statusMap["addresses"]
	if !ok {
		return nil // no addresses present. nothing to patch
	}

	addressList, ok := addresses.([]interface{})
	if !ok {
		return fmt.Errorf("unable to assert addresses into []interface{}. current values %v", addresses)
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	var newAddresses []interface{}
	for _, address := range addressList {
		addressMap, ok := address.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unable to assert address into map[string]string")
		}
		t := addressMap["type"]
		a := addressMap["address"]
		addressMap["address"] = "localhost"
		annotations[fmt.Sprintf("%soriginal-%s", simLabelPrefix, t)] = a.(string)
		newAddresses = append(newAddresses, addressMap)
	}

	statusMap["addresses"] = newAddresses
	obj.SetAnnotations(annotations)
	return unstructured.SetNestedField(obj.Object, statusMap, "status")
}
