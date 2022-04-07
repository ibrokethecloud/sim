package objects

import (
	wranglerunstructured "github.com/rancher/wrangler/pkg/unstructured"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

const (
	testDSFilePath      = "../../samples/sampleSupportBundle/yamls/namespaced/cattle-monitoring-system/apps/v1/daemonsets.yaml"
	testSTSFilePath     = "../../samples/sampleSupportBundle/yamls/namespaced/cattle-monitoring-system/apps/v1/statefulsets.yaml"
	testDeployFilePath  = "../../samples/sampleSupportBundle/yamls/namespaced/cattle-monitoring-system/apps/v1/deployments.yaml"
	testRSFilePath      = "../../samples/sampleSupportBundle/yamls/namespaced/cattle-monitoring-system/apps/v1/replicasets.yaml"
	testIngressFilePath = "../../samples/sampleSupportBundle/yamls/namespaced/harvester-system/extensions/v1beta1/ingresses.yaml"
	testSettingsPath    = "../../samples/sampleSupportBundle/yamls/cluster/management.cattle.io/v3/settings.yaml"
	testJobPath         = "../../samples/sampleSupportBundle/yamls/namespaced/harvester-system/batch/v1/jobs.yaml"
)

// TestParseDaemonSet will verify a sample Daemonset
func TestParseDaemonSetObject(t *testing.T) {
	// read our sample node-exporter daemonset in the samples directory
	verifyTestWorkloads(t, testDSFilePath)
}

// TestParseReplicaSetObject will verify a sample Replicaset
func TestParseReplicaSetObject(t *testing.T) {
	verifyTestWorkloads(t, testRSFilePath)
}

// TestParseStatefulSetObject will verify a sample StatefulSet
func TestParseStatefulSetObject(t *testing.T) {
	verifyTestWorkloads(t, testSTSFilePath)
}

// TestParseStatefulSetObject will verify a sample StatefulSet
func TestParseDeploymentObject(t *testing.T) {
	verifyTestWorkloads(t, testDeployFilePath)
}

func TestIngressObject(t *testing.T) {
	verifyTestWorkloads(t, testIngressFilePath)
}

func TestSettings(t *testing.T) {
	verifyTestWorkloads(t, testSettingsPath)
}

func verifyTestWorkloads(t *testing.T, path string) {
	objs, err := generateObjects(path)
	if err != nil {
		t.Fatalf("error reading sample daemonset file %s %v", testDSFilePath, err)
	}

	for _, obj := range objs {
		unstructObj, err := wranglerunstructured.ToUnstructured(obj)
		if err != nil {
			t.Fatalf("error converting runtime object to unstructured object %v", err)
		}

		t.Log("object pre cleanup")
		t.Logf("%v \n", unstructObj.Object)
		err = cleanupObjects(unstructObj.Object)
		if err != nil {
			t.Fatalf("error cleaning up objects %v", err)
		}

		t.Logf("%v \n", unstructObj.Object)

		if err != nil {
			t.Fatalf("error performing housekeeping on unstructured daemonset")
		}

		// verify objects
		ok, err := verifyObject(unstructObj, nil, "metadata", "resourceVersion")
		if err != nil {
			t.Fatalf("error verifying resource version object %v", err)
		}
		if ok {
			t.Fatalf("expected to find no resourceVersion but found one at metadata.resourceVersion")
		}

		ok, err = verifyObject(unstructObj, nil, "spec", "template", "metadata", "creationTimestamp")
		if err != nil {
			t.Fatalf("error verifying creationTimeStamp %v", err)
		}
		if ok {
			t.Fatalf("expected to find no creationTimeStamp but found one at spec.template.metadata.creationTimeStamp")
		}

		ok, err = verifyObject(unstructObj, func(v interface{}) (bool, error) {
			var volOk, innerOK bool
			var err error
			for _, volumes := range v.([]interface{}) {
				_, innerOK, err = unstructured.NestedFieldNoCopy(volumes.(map[string]interface{}), "hostPath", "type")
				if err != nil {
					return false, err
				}
				volOk = volOk || innerOK

			}
			return ok, err
		}, "spec", "template", "spec", "volumes")

		if err != nil {
			t.Fatalf("error verifying volumes in daemonset %v", err)
		}

		if ok {
			t.Fatalf("expected to find no type for hostPath volume definitions")
		}
	}
}

func TestVerifyJob(t *testing.T) {
	objs, err := generateObjects(testJobPath)
	if err != nil {
		t.Fatalf("error reading sample daemonset file %s %v", testDSFilePath, err)
	}
	for _, obj := range objs {
		unstructObj, err := wranglerunstructured.ToUnstructured(obj)
		if err != nil {
			t.Fatal(err)
		}

		err = cleanupObjects(unstructObj.Object)
		if err != nil {
			t.Fatal(err)
		}

		err = objectHousekeeping(unstructObj)
		if err != nil {
			t.Fatal(err)
		}
		labels, ok, err := unstructured.NestedStringMap(unstructObj.Object, "spec", "template", "metadata", "labels")
		if err != nil {
			t.Fatal(err)
		}

		if ok {
			t.Fatalf("expect to find no labels but found labels %v", labels)
		}
	}
}
