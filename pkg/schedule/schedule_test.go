package schedule

import (
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
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
		"accessMode-readonlyshared": {
			capacity: Capacity{
				Name: "access-test",
				AccessMode: &ResourceAccessMode{
					AllowReadOnlyShared: true,
					ReadOnlyShared:      3,
				},
			},
			request: CapacityRequest{
				Capacity:   "access-test",
				AccessMode: &ResourceAccessModeRequest{Request: ReadOnlyShared},
			},
			result: Capacity{
				Name: "access-test",
				AccessMode: &ResourceAccessMode{
					AllowReadOnlyShared: true,
					ReadOnlyShared:      4,
				},
			},
		},
		"accessMode-readwriteshared": {
			capacity: Capacity{
				Name: "access-test",
				AccessMode: &ResourceAccessMode{
					AllowReadWriteShared: true,
					ReadWriteShared:      3,
				},
			},
			request: CapacityRequest{
				Capacity:   "access-test",
				AccessMode: &ResourceAccessModeRequest{Request: ReadWriteShared},
			},
			result: Capacity{
				Name: "access-test",
				AccessMode: &ResourceAccessMode{
					AllowReadWriteShared: true,
					ReadWriteShared:      4,
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

func TestDeviceReduceCapacity(t *testing.T) {
	testCases := map[string]struct {
		device      Device
		allocations []CapacityResult
		result      Device
		expErr      string
	}{
		"missing capacity name for allocation": {
			device: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityResult{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "invalid-counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
			},
			expErr: `allocated capacity "invalid-counter-test" not found in device capacities`,
		},
		"missing capacity topology for allocation": {
			device: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityResult{
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
			expErr: `allocated capacity "counter-test;numa=numa-0" not found in device capacities`,
		},
		"single counter": {
			device: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:    "counter-test",
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityResult{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
			},
			result: Device{
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
			device: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name:     "quantity-test",
						Quantity: &ResourceQuantity{Capacity: resource.MustParse("10M")},
					},
				},
			},
			allocations: []CapacityResult{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "quantity-test",
						Quantity: &ResourceQuantityRequest{Request: resource.MustParse("1M")},
					},
				},
			},
			result: Device{
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
			device: Device{
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
			allocations: []CapacityResult{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "block-test",
						Quantity: &ResourceQuantityRequest{Request: resource.MustParse("1M")},
					},
				},
			},
			result: Device{
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
			device: Device{
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
			allocations: []CapacityResult{
				{
					CapacityRequest: CapacityRequest{
						Capacity: "counter-test",
						Counter:  &ResourceCounterRequest{Request: 4},
					},
				},
			},
			result: Device{
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
			device: Device{
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
			allocations: []CapacityResult{
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
			result: Device{
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
			device: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:          "numa",
								Name:          "numa-0",
								GroupInDevice: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityResult{
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
			result: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:          "numa",
								Name:          "numa-0",
								GroupInDevice: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 6},
					},
				},
			},
		},
		"single capacity, single topology type, multiple topologies": {
			device: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:          "numa",
								Name:          "numa-0",
								GroupInDevice: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:          "numa",
								Name:          "numa-1",
								GroupInDevice: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
				},
			},
			allocations: []CapacityResult{
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
			result: Device{
				Name: "test",
				Capacities: []Capacity{
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:          "numa",
								Name:          "numa-0",
								GroupInDevice: true,
							},
						},
						Counter: &ResourceCounter{Capacity: 10},
					},
					{
						Name: "counter-test",
						Topologies: []Topology{
							{
								Type:          "numa",
								Name:          "numa-1",
								GroupInDevice: true,
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
	basePool := DevicePool{
		Name:   "primary",
		Driver: "kubelet",
		Devices: []Device{
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
	singleAllocPool.Devices[0].Capacities[0].Counter.Capacity = 96

	testCases := map[string]struct {
		pool       DevicePool
		allocation PoolResult
		result     DevicePool
		expErr     string
	}{
		"single allocation": {
			pool: basePool,
			allocation: PoolResult{
				PoolName: "primary",
				DeviceResults: []DeviceResult{
					{
						DeviceName: "primary",
						CapacityResults: []CapacityResult{
							{
								CapacityRequest: CapacityRequest{
									Capacity: "pods",
									Counter:  &ResourceCounterRequest{Request: 4},
								},
							},
						},
					},
				},
				Best: 0,
			},
			result: singleAllocPool,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			result := tc.result
			err := result.ReduceCapacity(tc.allocation)
			if tc.expErr != "" {
				require.EqualError(t, err, tc.expErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.result, result)
			}
		})
	}
}
