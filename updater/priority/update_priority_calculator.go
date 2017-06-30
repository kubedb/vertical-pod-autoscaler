/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package priority

import (
	"math"
	"sort"

	"k8s.io/autoscaler/vertical-pod-autoscaler/updater/apimock"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/resource"
	apiv1 "k8s.io/kubernetes/pkg/api/v1"
)

const (
	// ignore change priority that is smaller than 10%
	defaultUpdateThreshod = 0.10
)

// UpdatePriorityCalculator is responsible for prioritizing updates on pods.
// It can returns a sorted list of pods in order of update priority.
// Update priority is proportional to fraction by which resources should be increased / decreased.
// i.e. pod with 10M current memory and recommendation 20M will have higher update priority
// than pod with pod with 100M current memory and and 150M recommendation (100% increase vs 50% increase)
type UpdatePriorityCalculator struct {
	resourcesPolicy *apimock.ResourcesPolicy
	cpuPolicy       *apimock.Policy
	pods            []podPriority
	config          *UpdateConfig
}

// UpdateConfig holds configuration for UpdatePriorityCalculator
type UpdateConfig struct {
	// MinChangePriority is the minimum change priority that will trigger a update.
	// TODO: should have separate for Mem and CPU?
	MinChangePriority float64
}

// NewUpdatePriorityCalculator creates new UpdatePriorityCalculator for the given resources policy and configuration.
// If the given policy is nil, there will be no policy restriction on update.
// If the given config is nil, default values are used.
func NewUpdatePriorityCalculator(policy *apimock.ResourcesPolicy, config *UpdateConfig) UpdatePriorityCalculator {
	if config == nil {
		config = &UpdateConfig{MinChangePriority: defaultUpdateThreshod}
	}
	return UpdatePriorityCalculator{resourcesPolicy: policy, config: config}
}

// AddPod adds pod to the UpdatePriorityCalculator.
func (calc *UpdatePriorityCalculator) AddPod(pod *apiv1.Pod, recommendation *apimock.Recommendation) {
	updatePriority := calc.getUpdatePriority(pod, recommendation)

	if updatePriority < calc.config.MinChangePriority {
		glog.V(2).Infof("pod not accepted for update %v, priority too low: %v", pod.Name, updatePriority)
		return
	}

	glog.V(2).Infof("pod accepted for update %v with priority %v", pod.Name, updatePriority)
	calc.pods = append(calc.pods, podPriority{pod, updatePriority})
}

// GetSortedPods returns a list of pods ordered by update priority (highest update priority first)
func (calc *UpdatePriorityCalculator) GetSortedPods() []*apiv1.Pod {
	sort.Sort(byPriority(calc.pods))

	result := make([]*apiv1.Pod, len(calc.pods))
	for i, podPrio := range calc.pods {
		result[i] = podPrio.pod
	}

	return result
}

func (calc *UpdatePriorityCalculator) getUpdatePriority(pod *apiv1.Pod, recommendation *apimock.Recommendation) float64 {
	var priority float64

	for _, podContainer := range pod.Spec.Containers {
		cr := getContainerRecommendation(podContainer.Name, recommendation)
		if cr == nil {
			glog.V(2).Infof("no recommendation for container %v in pod %v", podContainer.Name, pod.Name)
			continue
		}

		containerPolicy := getContainerPolicy(podContainer.Name, calc.resourcesPolicy)

		for resourceName, recommended := range cr.Resources {
			var (
				resourceRequested *resource.Quantity
				resourcePolicy    *apimock.Policy
			)

			if request, ok := podContainer.Resources.Requests[resourceName]; ok {
				resourceRequested = &request
			}
			if containerPolicy != nil {
				if policy, ok := (*containerPolicy)[resourceName]; ok {
					resourcePolicy = &policy
				}
			}
			resourceDiff := getPercentageDiff(resourceRequested, resourcePolicy, &recommended)
			priority += math.Abs(resourceDiff)
		}
	}
	return priority
}

func getPercentageDiff(request *resource.Quantity, policy *apimock.Policy, recommendation *resource.Quantity) float64 {
	if request == nil {
		// resource requirement is not currently specified
		// any recommendation for this resource we will treat as 100% change
		return 1.0
	}
	if recommendation == nil || recommendation.IsZero() {
		return 0
	}
	recommended := recommendation.Value()
	if policy != nil {
		if !policy.Min.IsZero() && recommendation.Value() < policy.Min.Value() {
			glog.Warningf("recommendation outside of policy bounds : min value : %v recommended : %v",
				policy.Min.Value(), recommended)
			recommended = policy.Min.Value()
		}
		if !policy.Max.IsZero() && recommendation.Value() > policy.Max.Value() {
			glog.Warningf("recommendation outside of policy bounds : max value : %v recommended : %v",
				policy.Max.Value(), recommended)
			recommended = policy.Max.Value()
		}
	}
	diff := recommended - request.Value()
	return float64(diff) / float64(request.Value())
}

func getContainerPolicy(containerName string, policy *apimock.ResourcesPolicy) *map[apiv1.ResourceName]apimock.Policy {
	if policy != nil {
		for _, container := range policy.Containers {
			if containerName == container.Name {
				return &container.ResourcePolicy
			}
		}
	}
	return nil
}

func getContainerRecommendation(containerName string, recommendation *apimock.Recommendation) *apimock.ContainerRecommendation {
	for _, container := range recommendation.Containers {
		if containerName == container.Name {
			return &container
		}
	}
	return nil
}

type podPriority struct {
	pod      *apiv1.Pod
	priority float64
}
type byPriority []podPriority

func (list byPriority) Len() int {
	return len(list)
}
func (list byPriority) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
func (list byPriority) Less(i, j int) bool {
	// reverse ordering, highest priority first
	return list[i].priority > list[j].priority
}
