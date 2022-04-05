package objects

import (
	"github.com/rancher/wrangler/pkg/unstructured"
	"testing"
)

// TestGenerateClusterScopedRuntimeObjects will test cluster scoped object generation from a sample support bundle
func TestGenerateClusterScopedRuntimeObjects(t *testing.T) {
	crds, clusterObjs, err := GenerateClusterScopedRuntimeObjects("../../samples/sampleSupportBundle")
	if err != nil {
		t.Fatalf("error processing crds and cluster scoped objects from support bundle %v", err)
	}

	t.Logf("found %d crds in support bundle", len(crds))
	t.Logf("found %d cluster scoped objects in support bundle", len(clusterObjs))
}

// TestGenerateNamepsacedRuntimeObjects will test namespaced cluster objects.
func TestGenerateNamespacedRuntimeObjects(t *testing.T) {
	nonpodObjs, podObjs, err := GenerateNamespacedRuntimeObjects("../../samples/sampleSupportBundle")
	if err != nil {
		t.Fatalf("error processing non namespaced objects and pods from support bundle %v", err)
	}

	// TODO: A better test with the objects.
	nonPodCount := 0
	for _, v := range nonpodObjs {
		nonPodCount += len(v)
	}
	t.Logf("found %d namespaced non-pod objects", nonPodCount)

	podCount := 0
	for _, v := range podObjs {
		podCount += len(v)
	}
	t.Logf("found %d namespaced pod objects", podCount)
}

// TestGenerateUnstructuredClusterObjects
func TestGenerateUnstructuredClusterObjects(t *testing.T) {
	crds, _, err := GenerateClusterScopedRuntimeObjects("../../samples/sampleSupportBundle")
	if err != nil {
		t.Fatalf("error processing crds and cluster scoped objects from support bundle %v", err)
	}

	unstructObj, err := unstructured.ToUnstructured(crds[0])
	if err != nil {
		t.Fatalf("error convering object to unstructured Obj %v", err)
	}
	t.Logf("group: %s version:%s kind:%s", unstructObj.GetObjectKind().GroupVersionKind().Group, unstructObj.GetObjectKind().GroupVersionKind().Version, unstructObj.GetObjectKind().GroupVersionKind().Kind)
	t.Log(unstructObj.GetObjectKind().GroupVersionKind().GroupVersion())
	t.Log(unstructObj.GetAPIVersion())
	t.Log(unstructObj.Object["spec"])
	t.Log(unstructObj.Object["status"])
}
