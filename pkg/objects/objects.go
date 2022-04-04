package objects

import "k8s.io/apimachinery/pkg/runtime"

// GenerateClusterScopedRuntimeObjects will parse the yaml directory
// in the bundle directory and apply the cluster and namespaced objects
func GenerateClusterScopedRuntimeObjects(path string) ([]runtime.Object, error) {
	var objs []runtime.Object

	return objs, nil
}
