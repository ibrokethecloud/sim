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
func GenerateNamespacedRuntimeObjects(path string) (nonpods map[string][]runtime.Object, pods map[string][]runtime.Object, err error) {

	var namespaces []string
	nonpods = make(map[string][]runtime.Object)
	pods = make(map[string][]runtime.Object)

	dir, err := filepath.Abs(filepath.Join(path, "yamls", "cluster"))
	if err != nil {
		return nonpods, pods, fmt.Errorf("error generating absolute path %v", err)
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v", path, err)
			return err
		}

		if info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			parent := filepath.Dir(absPath)
			// top level directory.. this is the list of namespaces we need
			if parent == dir {
				namespaces = append(namespaces, info.Name())
			}
		}

		return nil
	})

	if err != nil {
		return nonpods, pods, fmt.Errorf("error during dir walk %v", err)
	}

	// walk each namespace to get the runtime objects and populate the maps

	for _, ns := range namespaces {
		var nonpodObj, podObj []runtime.Object
		nsDir, err := filepath.Abs(filepath.Join(dir, ns))
		if err != nil {
			return nonpods, pods, err
		}
		err = filepath.Walk(nsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("prevent panic by handling failure accessing a path %q: %v", path, err)
				return err
			}

			if !info.IsDir() {
				if strings.Contains(info.Name(), "yaml") || strings.Contains(info.Name(), "yml") {
					absPath, err := filepath.Abs(path)
					if err != nil {
						return err
					}
					if strings.Contains(info.Name(), "pods") {
						// HERE!!!! Read files and add them to objects
						objs, err := generateObjects(absPath)
						if err != nil {
							return err
						}
						podObj = append(podObj, objs...)
					} else {
						objs, err := generateObjects(absPath)
						if err != nil {
							return err
						}
						nonpodObj = append(nonpodObj, objs...)
					}
				}
			}
			return nil
		})
		if err != nil {
			return nonpods, pods, fmt.Errorf("error during directory walk %v", err)
		}

		if len(nonpodObj) != 0 {
			nonpods[ns] = nonpodObj
		}

		if len(podObj) != 0 {
			pods[ns] = podObj
		}
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

}
