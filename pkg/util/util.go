package util

import (
	"fmt"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"path/filepath"
	"time"
)

func GeneratePodList(bundlePath string) ([]*v1.Pod, error) {
	podDetails := make(map[string][]string)
	// logs in support bundle are structured as
	// bundleRoot/logs
	// - namespaces/podname/logfile
	abs, err := filepath.Abs(filepath.Join(bundlePath, "logs"))
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
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						v1.Container{
							Name:  "supportbundle",
							Image: "supportbundle", //due to schedule logic nothing will be actually done
						},
					},
					DNSPolicy: v1.DNSClusterFirst,
					NodeSelector: map[string]string{
						"kubernetes.io/role": "agent",
						"type":               "virtual-kubelet",
					},
					Tolerations: []v1.Toleration{
						v1.Toleration{
							Key:      "virtual-kubelet.io/provider",
							Operator: v1.TolerationOpExists,
						},
					},
				},
			}
			podList = append(podList, tmpPod)
		}
	}
	return podList, nil
}

func DefaultPodStatus() *v1.PodStatus {
	podStatus := &v1.PodStatus{
		Phase: v1.PodRunning,
		Conditions: []v1.PodCondition{
			v1.PodCondition{
				Type:   v1.PodReady,
				Status: v1.ConditionTrue,
			},
			v1.PodCondition{
				Type:   v1.PodScheduled,
				Status: v1.ConditionTrue,
			},
			v1.PodCondition{
				Type:   v1.PodInitialized,
				Status: v1.ConditionTrue,
			},
			v1.PodCondition{
				Type:   v1.ContainersReady,
				Status: v1.ConditionTrue,
			},
		},
		ContainerStatuses: []v1.ContainerStatus{
			v1.ContainerStatus{
				Name:         "suportbundle",
				Ready:        true,
				RestartCount: 0,
				State: v1.ContainerState{
					Running: &v1.ContainerStateRunning{
						StartedAt: metav1.Time{Time: time.Now()},
					},
				},
			},
		},
		StartTime: &metav1.Time{Time: time.Now()},
	}
	return podStatus
}
