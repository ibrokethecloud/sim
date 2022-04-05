package objects

import (
	"bytes"
	"fmt"
	"github.com/rancher/wrangler/pkg/yaml"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path/filepath"
	"strings"
)

// GenerateClusterScopedRuntimeObjects will parse the yaml directory
// in the bundle directory and apply the cluster and namespaced objects
func GenerateClusterScopedRuntimeObjects(path string) (crd []runtime.Object, clusterObjs []runtime.Object, err error) {

	var crdList, noncrdList []string
	dir, err := filepath.Abs(filepath.Join(path, "yamls", "cluster"))
	if err != nil {
		return crd, clusterObjs, fmt.Errorf("error generating absolute path %v", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if strings.Contains(absPath, "apiextensions.k8s.io") {
				crdList = append(crdList, absPath)
			} else if strings.Contains(absPath, "admissionregistration.k8s.io") || strings.Contains(absPath, "metrics.k8s.io") || strings.Contains(absPath, "componentstatuses") {
				// TODO: Right now ignorning admission registration
				// as there are no actual running services and this breaks object processing.
				// TODO: Also ignoring metrics as we dont have API aggregation available
			} else {
				noncrdList = append(noncrdList, absPath)
			}

		}
		return nil
	})

	if err != nil {
		return crd, clusterObjs, fmt.Errorf("error during dir walk %v", err)
	}

	// generate objects //
	for _, v := range crdList {
		obj, err := generateObjects(v)
		if err != nil {
			return crd, clusterObjs, err
		}
		crd = append(crd, obj...)
	}

	for _, v := range noncrdList {
		obj, err := generateObjects(v)
		if err != nil {
			return crd, clusterObjs, err
		}
		clusterObjs = append(clusterObjs, obj...)
	}

	return crd, clusterObjs, nil
}

// GenerateNamespacedRuntimeObjects will return a map[string][]runtime.Object.
// the map key is the namespace and the list of objects associated with this namespaced.
// Two maps to split workloads into pods and nonpod types as pods may have dependency on other objects like service accounts.
func GenerateNamespacedRuntimeObjects(path string) (nonpods []runtime.Object, pods []runtime.Object, err error) {

	var podList, nonPodList []string
	dir, err := filepath.Abs(filepath.Join(path, "yamls", "namespaced"))
	if err != nil {
		return nonpods, pods, fmt.Errorf("error generating absolute path %v", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if !info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if strings.Contains(absPath, "pods.yaml") {
				podList = append(podList, absPath)
			} else {
				nonPodList = append(nonPodList, absPath)
			}

		}
		return nil
	})

	if err != nil {
		return nonpods, pods, fmt.Errorf("error during dir walk %v", err)
	}

	// walk each list to get the runtime objects and populate the maps
	// generate objects //
	for _, v := range podList {
		obj, err := generateObjects(v)
		if err != nil {
			return nonpods, pods, err
		}
		pods = append(pods, obj...)
	}

	for _, v := range nonPodList {
		obj, err := generateObjects(v)
		if err != nil {
			return nonpods, pods, err
		}
		nonpods = append(nonpods, obj...)
	}

	return nonpods, pods, err
}

func generateObjects(file string) (obj []runtime.Object, err error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return obj, err
	}

	obj, err = yaml.ToObjects(bytes.NewReader(content))
	return obj, err
	/*if err != nil {
		return obj, err
	}

	// additional conversion to unstructured type to allow wiping the resource version

	for _, o := range tmpObj {
		unstructuredObj, err := unstructured.ToUnstructured(o)
		if err != nil {
			return obj, err
		}
		unstructuredObj.SetResourceVersion("")
		obj = append(obj, unstructuredObj)
	}

	return obj, nil*/
}
