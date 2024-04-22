package schedule

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ptr[T any](val T) *T {
	var v T = val
	return &v
}

func genCapNumaNode(num int, cpu, mem resource.Quantity) []Capacity {
	return []Capacity{
		{
			Name:  "cpu",
			Block: &ResourceBlock{resource.MustParse("10m"), cpu},
			Topologies: []Topology{
				{
					Name:          fmt.Sprintf("numa-%d", num),
					Type:          "numa",
					GroupInDevice: true,
				},
			},
		},
		{
			Name:  "memory",
			Block: &ResourceBlock{resource.MustParse("1Mi"), mem},
			Topologies: []Topology{
				{
					Name:          fmt.Sprintf("numa-%d", num),
					Type:          "numa",
					GroupInDevice: true,
				},
			},
		},
	}
}

type numaGen struct {
	cpu, mem string
}

func genCapPrimaryPool(node, os, kernel, hw string, numa ...numaGen) DevicePool {
	capacities := []Capacity{
		{
			Name:    "pods",
			Counter: &ResourceCounter{100},
		},
		{
			Name:    "containers",
			Counter: &ResourceCounter{1000},
		},
	}
	for i, n := range numa {
		capacities = append(capacities, genCapNumaNode(i, resource.MustParse(n.cpu), resource.MustParse(n.mem))...)
	}

	return DevicePool{
		Driver: "kubelet",
		Name:   "primary",
		Attributes: []Attribute{
			{Name: "os", StringValue: &os},
			{Name: "kernel-release", SemVerValue: ptr(SemVer(kernel))},
			{Name: "hardware-platform", StringValue: &hw},
		},
		Devices: []Device{{
			Name:       "primary",
			Capacities: capacities,
		}},
	}
}

func genCapFoozerDevices(start, num int, model, version, conn, net, mem, foos string, vfs int64) []Device {
	var devices []Device
	for i := start; i < (start + num); i++ {
		topos := []Topology{
			{
				Name:          fmt.Sprintf("numa-%d", i/2),
				Type:          "numa",
				GroupInDevice: true,
			},
			{
				Name:          fmt.Sprintf("pci-%d", i%2),
				Type:          "pci",
				GroupInDevice: true,
			},
		}
		devices = append(devices, Device{
			Name: fmt.Sprintf("dev-foo-%d", i),
			Attributes: []Attribute{
				{Name: "model", StringValue: &model},
				{Name: "firmware-version", SemVerValue: ptr(SemVer(version))},
				{Name: "net-speed", QuantityValue: ptr(resource.MustParse(conn))},
			},
			Capacities: []Capacity{
				{
					Name:       "example.com/foozer/cores",
					Quantity:   &ResourceQuantity{resource.MustParse(foos)},
					Topologies: topos,
				},
				{
					Name:       "example.com/foozer/memory",
					Block:      &ResourceBlock{resource.MustParse("256Mi"), resource.MustParse(mem)},
					Topologies: topos,
				},
				{
					Name:    "example.com/foozer/interfaces",
					Counter: &ResourceCounter{vfs},
					Topologies: append(topos, Topology{
						Name:          net,
						Type:          "foo-net",
						GroupInDevice: true,
					}),
				},
			},
		})
	}
	return devices
}

// shape zero are compute nodes with no specialized devices
// They have 16 CPUs and 128Gi divided equally in two NUMA nodes
func GenCapShapeZero(num int) []NodeDevices {
	var nrs []NodeDevices
	for i := 0; i < num; i++ {
		node := fmt.Sprintf("shape-zero-%03d", i)
		nrs = append(nrs, NodeDevices{
			Name: node,
			Pools: []DevicePool{
				genCapPrimaryPool(node, "linux", "5.15.0-1046-gcp", "x86_64", numaGen{"8", "64Gi"}, numaGen{"8", "64Gi"}),
			},
		})
	}

	return nrs
}

// shape one consists of a node with 4 foozer-1000 cards
// the node has foozer kernel module/driver v7.8.1-gen6
// foozer 1000s only support node-local topology for their foo nets,
// so each node gets a separate foonet topology instance
func GenCapShapeOne(num int) []NodeDevices {
	pool := DevicePool{
		Driver: "example.com/foozer",
		Name:   "foozer-1000-01",
		Attributes: []Attribute{
			{Name: "driver-version", SemVerValue: ptr(SemVer("7.8.1-gen6"))},
		},
	}

	var nrs []NodeDevices
	for i := 0; i < num; i++ {
		node := fmt.Sprintf("shape-one-%03d", i)
		pool.Devices = genCapFoozerDevices(0, 4, "foozer-1000", "1.3.8", "10G", fmt.Sprintf("foonet-one-%03d", i), "64Gi", "8", 16)

		nrs = append(nrs, NodeDevices{
			Name: node,
			Pools: []DevicePool{
				genCapPrimaryPool(node, "linux", "5.15.0-1046-gcp", "x86_64", numaGen{"4", "32Gi"}, numaGen{"4", "32Gi"}),
				pool,
			},
		})
	}

	return nrs
}

// shape two consists of a node with 8 foozer-4000 cards
// the node requires a slightly different foozer kernel module/driver than shape one
// foozer 4000s support inter-node foonets, so there multiple nodes may be connected
// to a foonet topology. foozer-4000s have 40GB connections not 10GB
func GenCapShapeTwo(num, nets int) []NodeDevices {
	pool := DevicePool{
		Driver: "example.com/foozer",
		Name:   "foozer-4000-01",
		Attributes: []Attribute{
			{Name: "driver-version", SemVerValue: ptr(SemVer("7.8.2-gen8"))},
		},
	}
	var nrs []NodeDevices
	for i := 0; i < num; i++ {
		node := fmt.Sprintf("shape-two-%03d", i)
		pool.Devices = genCapFoozerDevices(0, 8, "foozer-4000", "1.8.8", "40G", fmt.Sprintf("foonet-two-%02d", i%nets), "256Gi", "16", 64)

		nrs = append(nrs, NodeDevices{
			Name: node,
			Pools: []DevicePool{
				genCapPrimaryPool(node, "linux", "5.15.0-1046-gcp", "x86_64", numaGen{"4", "32Gi"}, numaGen{"4", "32Gi"}),
				pool,
			},
		})
	}
	return nrs
}

// shape three consists of a mix 4 foozer-1000s and 4 foozer-4000s
func GenCapShapeThree(num, nets int) []NodeDevices {
	pool1 := DevicePool{
		Driver: "example.com/foozer",
		Name:   "foozer-1000-01",
		Attributes: []Attribute{
			{Name: "driver-version", SemVerValue: ptr(SemVer("7.8.2-gen8"))},
		},
	}

	pool2 := pool1
	pool2.Name = "foozer-4000-01"

	var nrs []NodeDevices
	for i := 0; i < num; i++ {
		node := fmt.Sprintf("shape-three-%03d", i)
		pool1.Devices = genCapFoozerDevices(0, 4, "foozer-1000", "1.3.8", "10G", fmt.Sprintf("foonet-three-%03d", i), "64Gi", "8", 16)
		pool2.Devices = genCapFoozerDevices(4, 4, "foozer-4000", "1.8.8", "40G", fmt.Sprintf("foonet-three-%02d", i%nets), "256Gi", "16", 64)

		nrs = append(nrs, NodeDevices{
			Name: fmt.Sprintf("shape-three-%03d", i),
			Pools: []DevicePool{
				genCapPrimaryPool(node, "linux", "5.15.0-1046-gcp", "x86_64", numaGen{"4", "32Gi"}, numaGen{"4", "32Gi"}),
				pool1,
				pool2,
			},
		})
	}

	return nrs
}

// claim generators

func genClaimFoozer(name, cores, mem string, vfs int64) DeviceClaim {
	return DeviceClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "example.k8s.io/v1alpha1",
			Kind:       "DeviceClaim",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: DeviceClaimSpec{
			Driver: ptr("example.com/foozer"),
			Requests: []CapacityRequest{
				{
					Capacity: "example.com/foozer/cores",
					Quantity: &ResourceQuantityRequest{Request: resource.MustParse(cores)},
				},
				{
					Capacity: "example.com/foozer/memory",
					Quantity: &ResourceQuantityRequest{Request: resource.MustParse(mem)},
				},
				{
					Capacity: "example.com/foozer/interfaces",
					Counter:  &ResourceCounterRequest{Request: vfs},
				},
			},
		},
	}
}
