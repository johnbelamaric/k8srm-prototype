package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestScheduleWithFullAvailability(t *testing.T) {
	testCases := map[string]struct {
		claim              CapacityClaim
		expectedAllocation *NodeCapacityAllocation
	}{
		"single pod, single container": {
			claim: CapacityClaim{
				Claims: []ResourceClaim{
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
	}

	capacity := genShapeZero(4)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			allocation := Schedule(capacity, &tc.claim)
			require.Equal(t, tc.expectedAllocation, allocation)
		})
	}
}
