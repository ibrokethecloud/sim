package supportbundle

import (
	"context"
	"fmt"
	"github.com/ibrokethecloud/sim/pkg/util"
	"github.com/virtual-kubelet/node-cli/manager"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"io"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"log"
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
	address         string
	port            int32
	config          *rest.Config
	client          *kubernetes.Clientset
}

// NewProvider creates a new Provider
func NewProvider(rm *manager.ResourceManager, nodeName, path, address string, port int32, config *rest.Config) (*Provider, error) {
	p := Provider{}
	var err error
	p.nodeName = nodeName
	p.operatingSystem = "linux"
	p.path = path
	p.address = address
	p.port = port
	p.config = config
	p.client, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
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

// GetPod returns the pod running in the cluster. returns nil
// if pod is not found.
func (p *Provider) GetPod(ctx context.Context, namespace, name string) (pod *v1.Pod, err error) {
	pod, err = p.client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	return pod, err
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *Provider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	contents, err := readLogFiles(p.path, namespace, podName, containerName)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(contents), nil
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
	// nothing to be done. Mark everything as running //
	pod, err := p.GetPod(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	return util.DefaultPodStatus(pod), nil
}

// GetPods returns a list of all pods known to be running in support bundleNode.
func (p *Provider) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	log.Printf("GetPods from path %s\n", p.path)

	return util.GeneratePodList(p.path)
}

// Capacity returns a resource list containing the capacity limits set for virtual-kubelet node.
func (p *Provider) Capacity(ctx context.Context) v1.ResourceList {
	return v1.ResourceList{
		"cpu":    resource.MustParse("20"),
		"memory": resource.MustParse("100Gi"),
		"pods":   resource.MustParse("200"),
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
	node.Status.Addresses = []v1.NodeAddress{
		v1.NodeAddress{
			Type:    v1.NodeInternalIP,
			Address: p.address,
		},
	}
	node.Status.DaemonEndpoints = v1.NodeDaemonEndpoints{
		KubeletEndpoint: v1.DaemonEndpoint{
			Port: p.port,
		},
	}
	node.Status.Capacity = p.Capacity(ctx)
	node.Status.Conditions = p.NodeConditions(ctx)
}
