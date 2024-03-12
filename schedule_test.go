package main

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestCapacityReduce(t *testing.T) {
	testCases := map[string]struct {
		capacity Capacity
		request  CapacityRequest
		result   Capacity
	}{
		"counter": {
			capacity: Capacity{
				Name:    "counter-test",
				Counter: &ResourceCounter{Capacity: 10},
			},
			request: CapacityRequest{
				Capacity: "counter-test",
				Counter:  &ResourceCounterRequest{Request: 4},
			},
			result: Capacity{
				Name:    "counter-test",
				Counter: &ResourceCounter{Capacity: 6},
			},
		},
		"quantity": {
			capacity: Capacity{
				Name:     "quantity-test",
				Quantity: &ResourceQuantity{Capacity: resource.MustParse("10M")},
			},
			request: CapacityRequest{
				Capacity: "quantity-test",
				Quantity: &ResourceQuantityRequest{Request: resource.MustParse("1M")},
			},
			result: Capacity{
				Name:     "quantity-test",
				Quantity: &ResourceQuantity{Capacity: resource.MustParse("9M")},
			},
		},
		"block": {
			capacity: Capacity{
				Name: "block-test",
				Block: &ResourceBlock{
					Capacity: resource.MustParse("10M"),
					Size:     resource.MustParse("1M"),
				},
			},
			request: CapacityRequest{
				Capacity: "block-test",
				Quantity: &ResourceQuantityRequest{Request: resource.MustParse("1M")},
			},
			result: Capacity{
				Name: "block-test",
				Block: &ResourceBlock{
					Capacity: resource.MustParse("9M"),
					Size:     resource.MustParse("1M"),
				},
			},
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			result, err := tc.capacity.reduce(tc.request)
			require.NoError(t, err)
			require.Equal(t, tc.result, result)
		})
	}

}

func TestResourceReduceCapacity(t *testing.T) {
	testCases := map[string]struct {
		resource    Resource
		allocations []CapacityAllocation
		result      Resource
		expErr      string
	}{
		"missing capacity name for allocation": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "invalid-counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
			},
			expErr: `allocated capacity "invalid-counter-test" not found in resource capacities`,
		},
		"missing capacity topology for allocation": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
					Topologies: []TopologyAssignment{
						{
							Type: "numa",
							Name: "numa-0",
						},
					},
				},
			},
			expErr: `allocated capacity "counter-test;numa=numa-0" not found in resource capacities`,
		},
		"single counter": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 6},
					},
				},
			},
		},
		"single quantity": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:     "quantity-test",
						Quantity: &ResourceQuantity{Capacity: resource.MustParse("10M")},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "quantity-test",
						Quantity: &ResourceQuantityRequest{Request: resource.MustParse("1M")},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:     "quantity-test",
						Quantity: &ResourceQuantity{Capacity: resource.MustParse("9M")},
					},
				},
			},
		},
		"single block": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "block-test",
						Block: &ResourceBlock{
							Capacity: resource.MustParse("10M"),
							Size:     resource.MustParse("1M"),
						},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "block-test",
						Quantity: &ResourceQuantityRequest{Request: resource.MustParse("1M")},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "block-test",
						Block: &ResourceBlock{
							Capacity: resource.MustParse("9M"),
							Size:     resource.MustParse("1M"),
						},
					},
				},
			},
		},
		"multiple capacities, one allocation": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
					{
						Name:    "counter-test-two",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 6},
					},
					{
						Name:    "counter-test-two",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
		},
		"multiple capacities, multiple allocations": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
					{
						Name:    "counter-test-two",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test-two",
						Counter:  &ResourceCounterRequest{Request: 1},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 6},
					},
					{
						Name:    "counter-test-two",
						Counter: &ResourceCounter{Capacity: 9},
					},
				},
			},
		},
		"single capacity with single topology": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:                "numa",
								Name:                "numa-0",
								AggregateInResource: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
					Topologies: []TopologyAssignment{
						{
							Type: "numa",
							Name: "numa-0",
						},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:                "numa",
								Name:                "numa-0",
								AggregateInResource: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 6},
					},
				},
			},
		},
		"single capacity, single topology type, multiple topologies": {
			resource: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:                "numa",
								Name:                "numa-0",
								AggregateInResource: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:                "numa",
								Name:                "numa-1",
								AggregateInResource: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityAllocation{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
					Topologies: []TopologyAssignment{
						{
							Type: "numa",
							Name: "numa-1",
						},
					},
				},
			},
			result: Resource{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:                "numa",
								Name:                "numa-0",
								AggregateInResource: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:                "numa",
								Name:                "numa-1",
								AggregateInResource: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 6},
					},
				},
			},
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			result := tc.result
			err := result.ReduceCapacity(tc.allocations)
			if tc.expErr != "" {
				require.EqualError(t, err, tc.expErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.result, result)
			}
		})
	}
}

func TestPoolReduceCapacity(t *testing.T) {
	basePool := ResourcePool{
		Name:   "primary",
		Driver: "kubelet",
		Resources: []Resource{
			{
				Name: "primary",
				Capacities: []Capacity{
					{
						Name:    "pods",
						Counter: &ResourceCounter{100},
					},
					{
						Name:    "containers",
						Counter: &ResourceCounter{1000},
					},
				},
			},
		},
	}

	singleAllocPool := basePool
	singleAllocPool.Resources[0].Capacities[0].Counter.Capacity = 96

	testCases := map[string]struct {
		pool       ResourcePool
		allocation PoolCapacityAllocation
		result     ResourcePool
		expErr     string
	}{
		"single allocation": {
			pool: basePool,
			allocation: PoolCapacityAllocation{
				PoolName:     "primary",
				ResourceName: "primary",
				CapacityAllocations: []CapacityAllocation{
					{
						CapacityRequest: CapacityRequest{
							Capacity: "pods",
							Counter:  &ResourceCounterRequest{Request: 4},
						},
					},
				},
			},
			result: singleAllocPool,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			result := tc.result
			err := result.ReduceCapacity(&tc.allocation)
			if tc.expErr != "" {
				require.EqualError(t, err, tc.expErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.result, result)
			}
		})
	}
}

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
		"single pod, multiple containers, with CPU and memory, enough": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name:   "my-pod",
					Claims: []ResourceClaim{genClaimPod()},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container-1",
						Claims: []ResourceClaim{genClaimContainer("7127m", "8Gi")},
					},
					{
						Name:   "my-container-2",
						Claims: []ResourceClaim{genClaimContainer("200m", "8Gi")},
					},
					{
						Name:   "my-container-3",
						Claims: []ResourceClaim{genClaimContainer("200m", "8Gi")},
					},
				},
			},
			expectSuccess: true,
		},
		"single pod, multiple containers, with CPU and memory, not enough": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name:   "my-pod",
					Claims: []ResourceClaim{genClaimPod()},
				},
				ContainerClaims: []CapacityClaim{
					{
						Name:   "my-container-1",
						Claims: []ResourceClaim{genClaimContainer("7127m", "8Gi")},
					},
					{
						Name:   "my-container-2",
						Claims: []ResourceClaim{genClaimContainer("8", "8Gi")},
					},
					{
						Name:   "my-container-3",
						Claims: []ResourceClaim{genClaimContainer("4", "8Gi")},
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

	for tn, tc := range testCases {
		capacity := genCapShapeZero(2)
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

	for tn, tc := range testCases {
		capacity := genCapShapeOne(2)
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

	for tn, tc := range testCases {
		capacity := genCapShapeTwo(2, 4)
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
