package objects

import (
	"context"
	"fmt"
	wranglerunstructured "github.com/rancher/wrangler/pkg/unstructured"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type ObjectManager struct {
	ctx    context.Context
	path   string
	config *rest.Config
	kc     *kubernetes.Clientset
	dc     dynamic.Interface
}

const (
	simCreationTimeStamp = "sim.harvesterhci.io/creationTimestamp"
)

// NewObjectManager is a wrapper around apply and support bundle path
func NewObjectManager(ctx context.Context, config *rest.Config, path string) (*ObjectManager, error) {

	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &ObjectManager{
		ctx:    ctx,
		path:   path,
		config: config,
		kc:     kc,
		dc:     dclient,
	}, nil
}

// CreateUnstructuredClusterObjects will use the dynamic client to create all
// cluster scoped objects from the support bundle
func (o *ObjectManager) CreateUnstructuredClusterObjects() error {
	crds, clusterObjs, err := GenerateClusterScopedRuntimeObjects(o.path)
	if err != nil {
		return err
	}

	// apply CRDs first
	err = o.applyObjects(crds, false, nil)
	if err != nil {
		return err
	}

	err = o.applyObjects(clusterObjs, false, &schema.GroupVersionResource{
		Group:    "apiregistration.k8s.io",
		Version:  "v1",
		Resource: "apiservices",
	})
	if err != nil {
		return err
	}
	return nil
}

// CreateUnstructuredObjects will use the dynamic client to create all
// namespace scoped objects from the support bundle
func (o *ObjectManager) CreateUnstructuredObjects() error {
	nonpods, pods, err := GenerateNamespacedRuntimeObjects(o.path)
	if err != nil {
		return err
	}

	// apply CRDs first
	err = o.applyObjects(nonpods, false, nil)
	if err != nil {
		return err
	}

	err = o.applyObjects(pods, false, nil)
	if err != nil {
		return err
	}
	return nil
}

// applyObjects is a wrapper to convert runtime.Objects to unstructured.Unstructured, perform some housekeeping before submitting the same to apiserver
func (o *ObjectManager) applyObjects(objs []runtime.Object, patchStatus bool, skipGVR *schema.GroupVersionResource) error {
	var dr dynamic.ResourceInterface
	var resp *unstructured.Unstructured
	for _, v := range objs {
		unstructuredObj, err := wranglerunstructured.ToUnstructured(v)
		if err != nil {
			return fmt.Errorf("error converting obj to unstructured %v", err)
		}

		err = objectHousekeeping(unstructuredObj)
		if err != nil {
			return fmt.Errorf("error during housekeeping on objects %v", err)
		}

		restMapping, err := findGVR(v.GetObjectKind().GroupVersionKind(), o.config)
		if err != nil {
			return fmt.Errorf("error looking up GVR %v", err)
		}

		if skipGVR != nil && restMapping.Resource == *skipGVR {
			continue
		}
		if restMapping.Scope.Name() == meta.RESTScopeNameNamespace {
			dr = o.dc.Resource(restMapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dr = o.dc.Resource(restMapping.Resource)
		}

		logrus.Infof("about to create resource %s with gvr %s", unstructuredObj.GetName(), restMapping.Resource.String())
		resp, err = dr.Create(o.ctx, unstructuredObj, metav1.CreateOptions{})
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				continue
			} else {
				return fmt.Errorf("error during creation of unstructured resource %v", err)
			}

		}

		if patchStatus {
			// we will patch the status here later
			logrus.Info(resp)
		}
	}
	return nil
}

// objectHousekeeping will perform some common housekeeping tasks based on GVR
// needed to keep the apiserver happy since we are dealing with exported CRDs
func objectHousekeeping(obj *unstructured.Unstructured) error {
	// Common housekeeping performed on all objects
	// need to clear resource version before apply.
	// this will be added as an annotation
	annotations, ok, err := unstructured.NestedStringMap(obj.Object, "metadata", "annotations")
	if err != nil {
		return err
	}
	if !ok {
		annotations = make(map[string]string)
	}

	resourceVersion, ok, err := unstructured.NestedString(obj.Object, "metadata", "resourceVersion")
	if err != nil {
		return err
	}
	if ok {
		annotations[simCreationTimeStamp] = resourceVersion
		unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
		err = unstructured.SetNestedStringMap(obj.Object, annotations, "metadata", "annotations")
		if err != nil {
			return err
		}
	}

	// GVR specific housekeeping
	if obj.GetKind() == "DaemonSet" || obj.GetKind() == "Deployment" || obj.GetKind() == "ReplicaSet" || obj.GetKind() == "StatefulSet" {
		// clear up any null timestamps from the template
		unstructured.RemoveNestedField(obj.Object, "spec", "template", "metadata", "creationTimestamp")
		// clear up type from hostPath volumes
		template, ok, err := unstructured.NestedFieldCopy(obj.Object, "spec", "template", "spec", "volumes")
		if err != nil {
			panic(err)
		}

		var cleanedVolumes []interface{}
		if ok {
			for _, v := range template.([]interface{}) {
				unstructured.RemoveNestedField(v.(map[string]interface{}), "hostPath", "type")
				cleanedVolumes = append(cleanedVolumes, v)
			}
		}
		err = unstructured.SetNestedField(obj.Object, cleanedVolumes, "spec", "template", "spec", "volumes")
	}

	// Ingresses are exported as extensions/v1beta1
	if obj.GetKind() == "Ingress" {
		obj.SetAPIVersion("networking.k8s.io/v1")
	}

	return nil
}

// wrapper to lookup GVR for usage with dynamic client
func findGVR(gvk schema.GroupVersionKind, cfg *rest.Config) (*meta.RESTMapping, error) {

	// DiscoveryClient queries API server about the resources
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	return mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
}

//verifyObj is a helper method used to verify objects to make it easier to test
func verifyObject(obj *unstructured.Unstructured, fn func(interface{}) (bool, error), keys ...string) (ok bool, err error) {
	tmpObj, ok, err := unstructured.NestedFieldCopy(obj.Object, keys...)
	// if not found or no key present
	if err != nil || !ok {
		return ok, err
	}

	return fn(tmpObj)
}
