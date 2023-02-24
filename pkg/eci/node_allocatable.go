package eci

import (
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	api_resource "k8s.io/apimachinery/pkg/api/resource"
)

func GetNodeAllocatableForECIPod(pod *v1.Pod) (v1.ResourceList, error) {
	cpu, ok := pod.Annotations[EciPodCpuLimitsAnno]
	if !ok {
		return nil, errors.New("no cpu limit")
	}
	cpuQuantity, err := api_resource.ParseQuantity(cpu)
	if err != nil {
		return nil, errors.Wrapf(err, "parse cpu limit (%s) failed", cpu)
	}

	memory, ok := pod.Annotations[EciPodMemLimitsAnno]
	if !ok {
		return nil, errors.New("no memory limit")
	}
	memory += "Gi"

	memQuantity, err := api_resource.ParseQuantity(memory)
	if err != nil {
		return nil, errors.Wrapf(err, "parse memory limit (%s) failed", memory)
	}

	resourcesList := make(v1.ResourceList)
	resourcesList[v1.ResourceCPU] = cpuQuantity
	resourcesList[v1.ResourceMemory] = memQuantity

	return resourcesList, nil
}
