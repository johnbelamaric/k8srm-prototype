package main

import (
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func TestSchedulePodForCore(t *testing.T) {
	testCases := map[string]struct {
		claim              PodCapacityClaim
		expectedAllocation *NodeCapacityAllocation
	}{
		"single pod, single container": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Claims: []ResourceClaim{genClaimPodContainer(1, 1)},
				},
			},
			expectedAllocation: &NodeCapacityAllocation{
				NodeName: "shape-zero-000",
				Allocations: []PoolCapacityAllocation{
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
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Claims: []ResourceClaim{genClaimPodContainer(1, 1)},
				},
				ContainerClaims: []CapacityClaim{
					{
						Claims: []ResourceClaim{genClaimCPUMem("7127m", "8Gi")},
					},
				},
			},
			expectedAllocation: &NodeCapacityAllocation{
				NodeName: "shape-zero-000",
				Allocations: []PoolCapacityAllocation{
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
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Claims: []ResourceClaim{
						genClaimPodContainer(1, 1),
						genClaimFoozer(1, "2Gi"),
					},
				},
			},
			expectedAllocation: nil,
		},
	}

	capacity := genCapShapeZero(4)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			allocation := SchedulePod(capacity, &tc.claim)
			require.Equal(t, tc.expectedAllocation, allocation)
		})
	}
}
