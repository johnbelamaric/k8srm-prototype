package main

import (
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

var rcOneContainerPod = ResourceClaim{
	Driver: "kubelet",
	Capacities: []CapacityRequest{
		{
			Capacity: "pods",
			Counter:  &ResourceCounterRequest{Request: 1},
		},
		{
			Capacity: "containers",
			Counter:  &ResourceCounterRequest{Request: 1},
		},
	},
}

var rcOneContainerPodCPUMem = ResourceClaim{
	Driver: "kubelet",
	Capacities: []CapacityRequest{
		{
			Capacity: "pods",
			Counter:  &ResourceCounterRequest{Request: 1},
		},
		{
			Capacity: "containers",
			Counter:  &ResourceCounterRequest{Request: 1},
		},
		{
			Capacity: "cpu",
			Quantity: &ResourceQuantityRequest{Request: resource.MustParse("7127m")},
		},
		{
			Capacity: "memory",
			Quantity: &ResourceQuantityRequest{Request: resource.MustParse("8Gi")},
		},
	},
}

var rcOneFooCore2GiFooMemory = ResourceClaim{
	Driver: "vendorFoo.com/foozer",
	Capacities: []CapacityRequest{
		{
			Capacity: "foo-cores",
			Counter:  &ResourceCounterRequest{Request: 1},
		},
		{
			Capacity: "foo-memory",
			Quantity: &ResourceQuantityRequest{Request: resource.MustParse("2Gi")},
		},
	},
}

func TestScheduleForCore(t *testing.T) {
	testCases := map[string]struct {
		claim              CapacityClaim
		expectedAllocation *NodeCapacityAllocation
	}{
		"single pod, single container": {
			claim: CapacityClaim{Core: rcOneContainerPod},
			expectedAllocation: &NodeCapacityAllocation{
				NodeName: "shape-zero-000",
				Allocations: []CapacityAllocation{
					{
						Driver: "kubelet",
						Capacities: []CapacityRequest{
							{
								Capacity: "pods",
								Counter:  &ResourceCounterRequest{Request: 1},
							},
							{
								Capacity: "containers",
								Counter:  &ResourceCounterRequest{Request: 1},
							},
						},
					},
				},
			},
		},
		"single pod, single container, with CPU and memory": {
			claim: CapacityClaim{Core: rcOneContainerPodCPUMem},
			expectedAllocation: &NodeCapacityAllocation{
				NodeName: "shape-zero-000",
				Allocations: []CapacityAllocation{
					{
						Driver: "kubelet",
						Capacities: []CapacityRequest{
							{
								Capacity: "pods",
								Counter:  &ResourceCounterRequest{Request: 1},
							},
							{
								Capacity: "containers",
								Counter:  &ResourceCounterRequest{Request: 1},
							},
							{
								Capacity: "cpu",
								Quantity: &ResourceQuantityRequest{Request: resource.MustParse("7130m")},
							},
							{
								Capacity: "memory",
								Quantity: &ResourceQuantityRequest{Request: resource.MustParse("8Gi")},
							},
						},
					},
				},
			},
		},
		"no resources for driver": {
			claim: CapacityClaim{
				Core:     rcOneContainerPod,
				Extended: []ResourceClaim{rcOneFooCore2GiFooMemory},
			},
			expectedAllocation: nil,
		},
	}

	capacity := genShapeZero(4)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			allocation := Schedule(capacity, &tc.claim)
			require.Equal(t, tc.expectedAllocation, allocation)
		})
	}
}
