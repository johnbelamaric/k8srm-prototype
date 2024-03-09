package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSchedulePodForCore(t *testing.T) {
	testCases := map[string]struct {
		claim              PodCapacityClaim
		expectSuccess      bool
	}{
		"single pod, single container": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Claims: []ResourceClaim{genClaimPodContainer(1, 1)},
				},
			},
			expectSuccess: true,
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
			expectSuccess: true,
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
