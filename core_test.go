package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSchedulePodForCore(t *testing.T) {
	testCases := map[string]struct {
		claim         PodCapacityClaim
		expectSuccess bool
	}{
		"single pod, single container": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name:   "my-pod",
					Claims: []ResourceClaim{genClaimPod()},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("", "")},
					},
				},
			},
			expectSuccess: true,
		},
		"single pod, single container, with CPU and memory": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name:   "my-pod",
					Claims: []ResourceClaim{genClaimPod()},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("7127m", "8Gi")},
					},
				},
			},
			expectSuccess: true,
		},
		"no resources for driver": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []ResourceClaim{
						genClaimPod(),
						genClaimFoozer("foozer", 1, "2Gi"),
					},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("7127m", "8Gi")},
					},
				},
			},
			expectSuccess: false,
		},
	}

	capacity := genCapShapeZero(4)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			allocation := SchedulePod(capacity, &tc.claim)
			require.Equal(t, tc.expectSuccess, allocation != nil)
		})
	}
}
