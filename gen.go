package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ptr[T any](val T) *T {
	var v T = val
	return &v
}

func genCoreNumaNode(num int, cpu, mem resource.Quantity) []Capacity {
	return []Capacity{
		{
			Name:  "cpus",
			Block: &ResourceBlock{resource.MustParse("10m"), cpu},
			Topologies: []Topology{
				{
					Name:      fmt.Sprintf("numa-%d", num),
					Type:      "numa",
					Aggregate: true,
				},
			},
		},
		{
			Name:  "memory",
			Block: &ResourceBlock{resource.MustParse("1Mi"), mem},
			Topologies: []Topology{{
				Name:      fmt.Sprintf("numa-%d", num),
				Type:      "numa",
				Aggregate: true},
			},
		},
	}
}

type numaGen struct {
	cpu, mem string
}

func genCorePool(os, kernel, hw string, numa ...numaGen) ResourcePool {
	var capacities []Capacity
	for i, n := range numa {
		capacities = append(capacities, genCoreNumaNode(i, resource.MustParse(n.cpu), resource.MustParse(n.mem))...)
	}

	return ResourcePool{
		Driver: "kubelet",
		Attributes: []Attribute{
			{Name: "os", StringValue: &os},
			{Name: "kernel-release", SemVerValue: ptr(SemVer(kernel))},
			{Name: "hardware-platform", StringValue: &hw},
		},
		Resources: []Resource{{
			Name:       "node",
			Capacities: capacities,
		}},
	}
}

func genFooResources(start, num int, model, version, conn, net, mem string, foos, vfs int64) []Resource {
	var resources []Resource
	for i := start; i < (start + num); i++ {
		resources = append(resources, Resource{
			Name: fmt.Sprintf("dev-foo-%d", i),
			Attributes: []Attribute{
				{Name: "model", StringValue: &model},
				{Name: "firmware-version", SemVerValue: ptr(SemVer(version))},
				{Name: "net-speed", QuantityValue: ptr(resource.MustParse(conn))},
			},
			Topologies: []Topology{
				{
					Name:      net,
					Type:      "foo-net",
					Aggregate: true,
				},
				{
					Name:      fmt.Sprintf("numa-%d", i/2),
					Type:      "numa",
					Aggregate: true,
				},
				{
					Name:      fmt.Sprintf("pci-%d", i%2),
					Type:      "pci",
					Aggregate: true,
				},
			},
			Capacities: []Capacity{
				{
					Name:    "foo-cores",
					Counter: &ResourceCounter{foos},
				},
				{
					Name:  "foo-memory",
					Block: &ResourceBlock{resource.MustParse("256Mi"), resource.MustParse(mem)},
				},
				{
					Name:    "vfs",
					Counter: &ResourceCounter{vfs},
				},
			},
		})
	}
	return resources
}

// shape zero are compute nodes with no specialized resources
// They have 16 CPUs and 128Gi divided equally in two NUMA nodes
func genShapeZero(num int) []NodeResources {
	core := genCorePool("linux", "5.15.0-1046-gcp", "x86_64", numaGen{"8", "64Gi"}, numaGen{"8", "64Gi"})

	var nrs []NodeResources
	for i := 0; i < num; i++ {
		nrs = append(nrs, NodeResources{
			Name: fmt.Sprintf("shape-zero-%03d", i),
			Core: core,
		})
	}

	return nrs
}

// shape one consists of a node with 4 foozer-1000 cards
// the node has foozer kernel module/driver v7.8.1-gen6
// foozer 1000s only support node-local topology for their foo nets,
// so each node gets a separate foonet topology instance
func genShapeOne(num int) []NodeResources {
	pool := ResourcePool{
		Driver: "vendorFoo.com/foozer",
		Attributes: []Attribute{
			{Name: "driver-version", SemVerValue: ptr(SemVer("7.8.1-gen6"))},
		},
	}

	core := genCorePool("linux", "5.15.0-1046-gcp", "x86_64", numaGen{"4", "32Gi"}, numaGen{"4", "32Gi"})

	var nrs []NodeResources
	for i := 0; i < num; i++ {

		pool.Resources = genFooResources(0, 4, "foozer-1000", "1.3.8", "10G", fmt.Sprintf("foonet-one-%03d", i), "64Gi", 8, 16)

		nrs = append(nrs, NodeResources{
			Name:     fmt.Sprintf("shape-one-%03d", i),
			Core:     core,
			Extended: []ResourcePool{pool},
		})
	}

	return nrs
}

// shape two consists of a node with 8 foozer-4000 cards
// the node requires a slightly different foozer kernel module/driver than shape one
// foozer 4000s support inter-node foonets, so there multiple nodes may be connected
// to a foonet topology. foozer-4000s have 40GB connections not 10GB
func genShapeTwo(num, nets int) []NodeResources {
	pool := ResourcePool{
		Driver: "vendorFoo.com/foozer",
		Attributes: []Attribute{
			{Name: "driver-version", SemVerValue: ptr(SemVer("7.8.2-gen8"))},
		},
	}
	core := genCorePool("linux", "5.15.0-1046-gcp", "x86_64", numaGen{"4", "32Gi"}, numaGen{"4", "32Gi"})
	var nrs []NodeResources
	for i := 0; i < num; i++ {

		pool.Resources = genFooResources(0, 8, "foozer-4000", "1.8.8", "40G", fmt.Sprintf("foonet-two-%02d", i%nets), "256Gi", 16, 64)

		nrs = append(nrs, NodeResources{
			Name:     fmt.Sprintf("shape-two-%03d", i),
			Core:     core,
			Extended: []ResourcePool{pool},
		})
	}
	return nrs
}

// shape three consists of a mix 4 foozer-1000s and 4 foozer-4000s
func genShapeThree(num, nets int) []NodeResources {
	pool := ResourcePool{
		Driver: "vendorFoo.com/foozer",
		Attributes: []Attribute{
			{Name: "driver-version", SemVerValue: ptr(SemVer("7.8.2-gen8"))},
		},
	}

	core := genCorePool("linux", "5.15.0-1046-gcp", "x86_64", numaGen{"4", "32Gi"}, numaGen{"4", "32Gi"})

	var nrs []NodeResources
	for i := 0; i < num; i++ {
		pool.Resources = genFooResources(0, 4, "foozer-1000", "1.3.8", "10G", fmt.Sprintf("foonet-three-%03d", i), "64Gi", 8, 16)
		pool.Resources = append(pool.Resources, genFooResources(4, 4, "foozer-4000", "1.8.8", "40G", fmt.Sprintf("foonet-three-%02d", i%nets), "256Gi", 16, 64)...)

		nrs = append(nrs, NodeResources{
			Name:     fmt.Sprintf("shape-three-%03d", i),
			Core:     core,
			Extended: []ResourcePool{pool},
		})
	}

	return nrs
}
