package objects

import (
	"context"
	"fmt"
	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/apply"
	apiadmissionv1 "github.com/rancher/wrangler/pkg/generated/controllers/admissionregistration.k8s.io/v1"
	apiextensionsv1 "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io/v1"
	apiregistrationv1 "github.com/rancher/wrangler/pkg/generated/controllers/apiregistration.k8s.io/v1"
	v1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	wranglerunstructured "github.com/rancher/wrangler/pkg/unstructured"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/util/workqueue"
	metrics "k8s.io/metrics/pkg/apis/metrics/install"
	"time"
)

type ObjectManager struct {
	ctx context.Context
	apply.Apply
	path   string
	config *rest.Config
	kc     *kubernetes.Clientset
	dc     dynamic.Interface
}

func init() {
	metrics.Install(scheme.Scheme)
}

// NewObjectManager is a wrapper around apply and support bundle path
func NewObjectManager(ctx context.Context, config *rest.Config, path string) (*ObjectManager, error) {

	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 5*time.Minute)
	workqueue.DefaultControllerRateLimiter()
	clientFactory, err := client.NewSharedClientFactory(config, nil)
	if err != nil {
		return nil, err
	}

	cacheFactory := cache.NewSharedCachedFactory(clientFactory, nil)

	scf := controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultRateLimiter: rateLimit,
		DefaultWorkers:     5,
	})

	cmController := v1.NewConfigMapController(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, "configmap", true, scf)

	apiExtensionsController := apiextensionsv1.NewCustomResourceDefinitionController(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}, "apiextensions", false, scf)

	validatingWebhookController := apiadmissionv1.NewValidatingWebhookConfigurationController(schema.GroupVersionKind{
		Group:   "admissionregistration.k8s.io",
		Version: "v1",
		Kind:    "ValidatingWebhookConfiguration",
	}, "validatingwebhook", false, scf)

	mutatingWebhookController := apiadmissionv1.NewValidatingWebhookConfigurationController(schema.GroupVersionKind{
		Group:   "admissionregistration.k8s.io",
		Version: "v1",
		Kind:    "MutatingWebhookConfiguration",
	}, "mutatingwebhook", false, scf)

	apiService := apiregistrationv1.NewAPIServiceController(schema.GroupVersionKind{
		Group:   "apiregistration.k8s.io",
		Version: "v1",
		Kind:    "APIService",
	}, "apiservice", false, scf)

	a := apply.New(discovery, apply.NewClientFactory(config), cmController, apiExtensionsController, validatingWebhookController, mutatingWebhookController, apiService)
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &ObjectManager{
		ctx:    ctx,
		Apply:  a,
		path:   path,
		config: config,
		kc:     kc,
		dc:     dclient,
	}, nil
}

// CreateClusterScopedObjects will parse all cluster scoped objects and
// create them
func (o *ObjectManager) CreateClusterScopedObjects() error {
	crds, clusterObjs, err := GenerateClusterScopedRuntimeObjects(o.path)
	if err != nil {
		return err
	}

	clusterObjCm, err := o.createOwnerCM("cluster-owner")
	if err != nil {
		return fmt.Errorf("error creating cluster owner cm %v", err)
	}

	err = o.WithContext(o.ctx).WithOwner(clusterObjCm).ApplyObjects(crds...)
	if err != nil {
		return fmt.Errorf("error during CRD application %v", err)
	}

	err = o.WithContext(o.ctx).WithOwner(clusterObjCm).ApplyObjects(clusterObjs...)
	if err != nil {
		return fmt.Errorf("error durnig Cluster scoped object application %v", err)
	}

	return nil
}

// checkAndCreateNamespace will check and create a namespace if needed
func (o *ObjectManager) checkAndCreateNamespace(ns string) error {
	_, err := o.kc.CoreV1().Namespaces().Get(o.ctx, ns, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ns,
				},
			}
			_, err = o.kc.CoreV1().Namespaces().Create(o.ctx, namespace, metav1.CreateOptions{})
			return err
		} else {
			return err
		}
	}

	return nil
}

func (o *ObjectManager) createOwnerCM(name string) (*corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}
	return o.kc.CoreV1().ConfigMaps("default").Create(o.ctx, &cm, metav1.CreateOptions{})
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

		// need to clear resource version before apply
		unstructured.RemoveNestedField(unstructuredObj.Object, "metadata", "resourceVersion")
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
