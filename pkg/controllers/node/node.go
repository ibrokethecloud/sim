package node

import (
	"context"
	"fmt"

	corecontrollers "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

const (
	patchKey          = "sim.harvesterhci.io/node-controller"
	nodeAddressPrefix = "sim.harvesterhci.io/node-address"
)

type handler struct {
	ctx  context.Context
	node corecontrollers.NodeController
}

func Register(ctx context.Context, node corecontrollers.NodeController) {
	h := &handler{
		ctx:  ctx,
		node: node,
	}
	node.OnChange(ctx, "node-change", h.PatchNodes)
}

func (h *handler) PatchNodes(key string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil || len(node.Status.Addresses) == 0 {
		return node, nil
	}

	// if patch key is present check and update addresses
	if _, ok := node.Annotations[patchKey]; ok {
		return node, nil
	}

	// patch nodes
	var annotations map[string]string
	if node.Annotations != nil {
		annotations = node.GetAnnotations()
	} else {
		annotations = make(map[string]string)
	}

	var newAddresses []corev1.NodeAddress
	for k, v := range node.Status.Addresses {
		if v.Address != "localhost" {
			annotations[fmt.Sprintf("%s-%d", nodeAddressPrefix, k)] = v.String()
			newAddresses = append(newAddresses, corev1.NodeAddress{
				Type:    v.Type,
				Address: "localhost",
			})
		}
	}

	var labels map[string]string
	if node.Labels != nil {
		labels = node.GetLabels()
	} else {
		labels = make(map[string]string)
	}
	labels[patchKey] = "true"

	if len(newAddresses) != 0 {
		node.SetAnnotations(annotations)
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		newObj, err := h.node.Get(key, metav1.GetOptions{})
		if err != nil {
			return err
		}
		newObj.Annotations = node.Annotations
		newObj.SetLabels(labels)
		_, err = h.node.Update(newObj)
		return err
	})

	if err != nil {
		return node, err
	}

	if len(newAddresses) != 0 {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			newObj, err := h.node.Get(key, metav1.GetOptions{})
			if err != nil {
				return err
			}
			newObj.Status.Addresses = newAddresses
			_, err = h.node.UpdateStatus(newObj)
			return err
		})
	}

	return node, nil

}
