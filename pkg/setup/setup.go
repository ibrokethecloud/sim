package setup

import (
	"context"
	"github.com/ibrokethecloud/sim/pkg/util"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// SetupPodObjects will parse the support bundle and create pods scheduled to virtual kubelet
// virtual kubelet will know what to do next...
func SetupPodObjects(ctx context.Context, config *rest.Config, bundlePath string) error {
	podList, err := util.GeneratePodList(bundlePath)
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	for _, p := range podList {
		err = checkAndCreateNamespace(ctx, client, p.Namespace)
		if err != nil {
			return err
		}
		err = checkAndCreatePod(ctx, client, p)
		if err != nil {
			return err
		}
	}
	return nil
}

func checkAndCreateNamespace(ctx context.Context, client *kubernetes.Clientset, namespace string) error {
	if namespace == "" {
		return nil // default namespace
	}

	_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}, metav1.CreateOptions{})
			return err
		} else {
			return err
		}
	}

	return nil
}

func checkAndCreatePod(ctx context.Context, client *kubernetes.Clientset, p *v1.Pod) error {
	var err error
	_, err = client.CoreV1().Pods(p.Namespace).Get(ctx, p.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.CoreV1().Pods(p.Namespace).Create(ctx, p, metav1.CreateOptions{})
			return err
		}
	}

	return err
}
