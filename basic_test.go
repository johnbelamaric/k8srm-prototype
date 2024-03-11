package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
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
		"single pod, single container, with CPU and memory, enough": {
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
		"single pod, single container, with CPU and memory, insufficient CPU": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name:   "my-pod",
					Claims: []ResourceClaim{genClaimPod()},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("64", "8Gi")},
					},
				},
			},
			expectSuccess: false,
		},
		"single pod, single container, with CPU and memory, insufficient memory": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name:   "my-pod",
					Claims: []ResourceClaim{genClaimPod()},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("4", "128Gi")},
					},
				},
			},
			expectSuccess: false,
		},
		"no resources for driver": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []ResourceClaim{
						genClaimPod(),
						genClaimFoozer("foozer", "1m", "2Gi", 1),
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

	capacity := genCapShapeZero(2)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			fmt.Println("-------------------------------")
			fmt.Println(tn)
			fmt.Println("----")
			allocation := SchedulePod(capacity, &tc.claim)
			require.Equal(t, tc.expectSuccess, allocation != nil)
			fmt.Println("----")
			b, _ := yaml.Marshal(allocation)
			fmt.Println(string(b))
		})
	}
}

func TestSchedulePodForFoozer(t *testing.T) {
	testCases := map[string]struct {
		claim         PodCapacityClaim
		expectSuccess bool
	}{
		"single pod, container, cpu/mem, and foozer": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []ResourceClaim{
						genClaimPod(),
						genClaimFoozer("foozer", "1", "2Gi", 0),
					},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("1", "4Gi")},
					},
				},
			},
			expectSuccess: true,
		},
		"no foozer big enough": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []ResourceClaim{
						genClaimPod(),
						genClaimFoozer("foozer", "16", "32Gi", 0),
					},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("1", "4Gi")},
					},
				},
			},
			expectSuccess: false,
		},
	}

	capacity := genCapShapeOne(2)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			fmt.Println("-------------------------------")
			fmt.Println(tn)
			fmt.Println("----")
			allocation := SchedulePod(capacity, &tc.claim)
			require.Equal(t, tc.expectSuccess, allocation != nil)
			fmt.Println("----")
			b, _ := yaml.Marshal(allocation)
			fmt.Println(string(b))
		})
	}
}

func TestSchedulePodForBigFoozer(t *testing.T) {
	testCases := map[string]struct {
		claim         PodCapacityClaim
		expectSuccess bool
	}{
		"single pod, container, cpu/mem, and foozer": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []ResourceClaim{
						genClaimPod(),
						genClaimFoozer("foozer", "1", "2Gi", 0),
					},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("1", "4Gi")},
					},
				},
			},
			expectSuccess: true,
		},
		"no foozer big enough": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []ResourceClaim{
						genClaimPod(),
						genClaimFoozer("foozer", "16", "32Gi", 0),
					},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container",
						Claims: []ResourceClaim{genClaimContainer("1", "4Gi")},
					},
				},
			},
			expectSuccess: true,
		},
	}

	capacity := genCapShapeTwo(2,4)
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			fmt.Println("-------------------------------")
			fmt.Println(tn)
			fmt.Println("----")
			allocation := SchedulePod(capacity, &tc.claim)
			require.Equal(t, tc.expectSuccess, allocation != nil)
			fmt.Println("----")
			b, _ := yaml.Marshal(allocation)
			fmt.Println(string(b))
		})
	}
}

