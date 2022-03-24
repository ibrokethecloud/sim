package supportbundle

import (
	"context"
	"fmt"
	"github.com/virtual-kubelet/node-cli/manager"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// SupportBundle provider constants
const (
	jobNamePrefix           = "supportbundle-virtual-kubelet"
	supportBundleAnnotation = "harvesterhci.io/supportbundle"
)

// Provider implements the virtual-kubelet provider interface and loads support bundle
type Provider struct {
	nodeName        string
	operatingSystem string
	path            string
}

// NewProvider creates a new Provider
func NewProvider(rm *manager.ResourceManager, nodeName, operatingSystem, path string) (*Provider, error) {
	p := Provider{}
	p.nodeName = nodeName
	p.operatingSystem = operatingSystem
	p.path = path
	return &p, nil
}

// CreatePod is a noop. Existing bundles will not be modified
func (p *Provider) CreatePod(ctx context.Context, pod *v1.Pod) error {
	log.Println("Pod Create called: No-op as not implemented")
	return nil
}

// UpdatePod is a noop. Existing bundles will not be modified
func (p *Provider) UpdatePod(ctx context.Context, pod *v1.Pod) error {
	log.Println("Pod Update called: No-op as not implemented")
	return nil
}

// DeletePod is a noop. Existing bundles will not be modified
func (p *Provider) DeletePod(ctx context.Context, pod *v1.Pod) (err error) {
	log.Println("Pod Create called: No-op as not implemented")

	return nil
}

// GetPod returns the pod running in the Nomad cluster. returns nil
// if pod is not found.
func (p *Provider) GetPod(ctx context.Context, namespace, name string) (pod *v1.Pod, err error) {
	return pod, nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *Provider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader("")), nil
}

// GetPodFullName as defined in the provider context
func (p *Provider) GetPodFullName(ctx context.Context, namespace string, pod string) string {
	return fmt.Sprintf("%s-%s", jobNamePrefix, pod)
}

// RunInContainer is a noop. There is no actual compute backing this object
func (p *Provider) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
	return nil
}

// GetPodStatus returns the status of a pod by name that is running as a job
func (p *Provider) GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	podStatus := &v1.PodStatus{
		Phase: v1.PodRunning,
	}
	return podStatus, nil
}

// GetPods returns a list of all pods known to be running in support bundleNode.
func (p *Provider) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	log.Printf("GetPods from path %s\n", p.path)

	podDetails := make(map[string][]string)
	// logs in support bundle are structured as
	// bundleRoot/logs
	// - namespaces/podname/logfile
	abs, err := filepath.Abs(filepath.Join(p.path, "logs"))
	if err != nil {
		return nil, err
	}

	// populate parent folders in logs
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if info.IsDir() {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			parent := filepath.Dir(absPath)
			if parent == abs {
				filepath.Join(abs, info.Name())
				podDetails[filepath.Join(abs, info.Name())] = []string{}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// populate list of sub directories
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		if info.IsDir() {
			parent := filepath.Dir(path)
			absParent, err := filepath.Abs(parent)
			if err != nil {
				return err
			}
			if _, ok := podDetails[absParent]; ok {
				podDetails[absParent] = append(podDetails[absParent], info.Name())
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Println(podDetails)
	podList := []*v1.Pod{}
	for parent, values := range podDetails {
		base := filepath.Base(parent)
		for _, v := range values {
			tmpPod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      v,
					Namespace: base,
				},
			}
			podList = append(podList, tmpPod)
		}
	}
	return podList, nil
}

// Capacity returns a resource list containing the capacity limits set for Nomad.
func (p *Provider) Capacity(ctx context.Context) v1.ResourceList {
	// TODO: Use nomad /nodes api to get a list of nodes in the cluster
	// and then use the read node /node/:node_id endpoint to calculate
	// the total resources of the cluster to report back to kubernetes.
	return v1.ResourceList{
		"cpu":    resource.MustParse("20"),
		"memory": resource.MustParse("100Gi"),
		"pods":   resource.MustParse("20"),
	}
}

// NodeConditions returns a list of conditions (Ready, OutOfDisk, etc), for updates to the node status
// within Kubernetes.
func (p *Provider) NodeConditions(ctx context.Context) []v1.NodeCondition {
	// TODO: Make these dynamic.
	return []v1.NodeCondition{
		{
			Type:               "Ready",
			Status:             v1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "kubelet is ready.",
		},
		{
			Type:               "OutOfDisk",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientDisk",
			Message:            "kubelet has sufficient disk space available",
		},
		{
			Type:               "MemoryPressure",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasSufficientMemory",
			Message:            "kubelet has sufficient memory available",
		},
		{
			Type:               "DiskPressure",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "kubelet has no disk pressure",
		},
		{
			Type:               "NetworkUnavailable",
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "RouteCreated",
			Message:            "RouteController created a route",
		},
	}

}

// NodeAddresses returns a list of addresses for the node status
// within Kubernetes.
func (p *Provider) NodeAddresses(ctx context.Context) []v1.NodeAddress {

	return []v1.NodeAddress{
		v1.NodeAddress{
			Address: "localhost",
			Type:    v1.NodeHostName,
		},
	}
}

// NodeDaemonEndpoints returns NodeDaemonEndpoints for the node status
// within Kubernetes.
func (p *Provider) NodeDaemonEndpoints(ctx context.Context) *v1.NodeDaemonEndpoints {
	return &v1.NodeDaemonEndpoints{KubeletEndpoint: v1.DaemonEndpoint{
		Port: 8080,
	}}
}

// OperatingSystem returns the operating system for this provider.
// This is a noop to default to Linux for now.
func (p *Provider) OperatingSystem() string {
	return p.operatingSystem
}

// ConfigureNode enables a provider to configure the node object that
// will be used for Kubernetes.
func (p *Provider) ConfigureNode(ctx context.Context, node *v1.Node) {
	// do nothing yet
	node.Status.Capacity = p.Capacity(ctx)
	node.Status.Conditions = p.NodeConditions(ctx)
}
