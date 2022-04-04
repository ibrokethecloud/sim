package objects

import (
	"context"
	"fmt"
	"github.com/rancher/wrangler/pkg/apply"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ObjectManager struct {
	ctx context.Context
	apply.Apply
	path   string
	config *rest.Config
	kc     *kubernetes.Clientset
}

// NewObjectManager is a wrapper around apply and support bundle path
func NewObjectManager(ctx context.Context, config *rest.Config, path string) (*ObjectManager, error) {
	a, err := apply.NewForConfig(config)
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
	}, nil
}

// CreateClusterScopedObjects will parse all cluster scoped objects and
// create them
func (o *ObjectManager) CreateClusterScopedObjects() error {
	crds, clusterObjs, err := GenerateClusterScopedRuntimeObjects(o.path)
	if err != nil {
		return err
	}

	err = o.WithContext(o.ctx).ApplyObjects(crds...)
	if err != nil {
		return fmt.Errorf("error during CRD application %v", err)
	}

	err = o.WithContext(o.ctx).ApplyObjects(clusterObjs...)
	if err != nil {
		return fmt.Errorf("error durnig Cluster scoped object application %v", err)
	}

	return nil
}

func (o *ObjectManager) CreateNamespacedObjects() error {
	nonpods, pods, err := GenerateNamespacedRuntimeObjects(o.path)
	if err != nil {
		return err
	}

	for ns, obj := range nonpods {
		err = o.checkAndCreateNamespace(ns)
		if err != nil {
			return fmt.Errorf("error during creating namespace %v", err)
		}
		err = o.WithContext(o.ctx).WithDefaultNamespace(ns).ApplyObjects(obj...)
		if err != nil {
			return fmt.Errorf("errur during namespaced non-pod object apply %v", err)
		}
	}

	for ns, obj := range pods {
		err = o.checkAndCreateNamespace(ns)
		if err != nil {
			return fmt.Errorf("error during creating namespace %v", err)
		}
		err = o.WithContext(o.ctx).WithDefaultNamespace(ns).ApplyObjects(obj...)
		if err != nil {
			return fmt.Errorf("errur during namespaced non-pod object apply %v", err)
		}
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
