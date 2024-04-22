# k8srm-prototype

For more background, please see this document, though it is not yet up to date with the latest in this repo:
- [Revisiting Kubernetes Resource Model](https://docs.google.com/document/d/1Xy8HpGATxgA2S5tuFWNtaarw5KT8D2mj1F4AP1wg6dM/edit?usp=sharing).


## Building

Just run `make`, it will build everything.

```console
k8srm-prototype$ make
gofmt -s -w .
go test -v ./...
?   	github.com/johnbelamaric/k8srm-prototype/cmd/mock-apiserver	[no test files]
?   	github.com/johnbelamaric/k8srm-prototype/cmd/schedule	[no test files]
=== RUN   TestMeetsConstraints
=== RUN   TestMeetsConstraints/simple_device_constraint_met
=== RUN   TestMeetsConstraints/simple_device_constraint_failed
=== RUN   TestMeetsConstraints/simple_device_and_pool_constraint_met
=== RUN   TestMeetsConstraints/simple_device_and_pool_constraint_failed
=== RUN   TestMeetsConstraints/quantity_constraint_met
=== RUN   TestMeetsConstraints/nil_constraint
=== RUN   TestMeetsConstraints/empty_constraint
--- PASS: TestMeetsConstraints (0.00s)
    --- PASS: TestMeetsConstraints/simple_device_constraint_met (0.00s)
    --- PASS: TestMeetsConstraints/simple_device_constraint_failed (0.00s)
    --- PASS: TestMeetsConstraints/simple_device_and_pool_constraint_met (0.00s)
    --- PASS: TestMeetsConstraints/simple_device_and_pool_constraint_failed (0.00s)
    --- PASS: TestMeetsConstraints/quantity_constraint_met (0.00s)
    --- PASS: TestMeetsConstraints/nil_constraint (0.00s)
    --- PASS: TestMeetsConstraints/empty_constraint (0.00s)
=== RUN   TestCapacityReduce
=== RUN   TestCapacityReduce/counter
=== RUN   TestCapacityReduce/quantity
=== RUN   TestCapacityReduce/block
=== RUN   TestCapacityReduce/accessMode-readonlyshared
=== RUN   TestCapacityReduce/accessMode-readwriteshared
--- PASS: TestCapacityReduce (0.00s)
    --- PASS: TestCapacityReduce/counter (0.00s)
    --- PASS: TestCapacityReduce/quantity (0.00s)
    --- PASS: TestCapacityReduce/block (0.00s)
    --- PASS: TestCapacityReduce/accessMode-readonlyshared (0.00s)
    --- PASS: TestCapacityReduce/accessMode-readwriteshared (0.00s)
=== RUN   TestDeviceReduceCapacity
=== RUN   TestDeviceReduceCapacity/multiple_capacities,_multiple_allocations
=== RUN   TestDeviceReduceCapacity/single_capacity_with_single_topology
=== RUN   TestDeviceReduceCapacity/single_capacity,_single_topology_type,_multiple_topologies
=== RUN   TestDeviceReduceCapacity/missing_capacity_topology_for_allocation
=== RUN   TestDeviceReduceCapacity/single_quantity
=== RUN   TestDeviceReduceCapacity/multiple_capacities,_one_allocation
=== RUN   TestDeviceReduceCapacity/missing_capacity_name_for_allocation
=== RUN   TestDeviceReduceCapacity/single_counter
=== RUN   TestDeviceReduceCapacity/single_block
--- PASS: TestDeviceReduceCapacity (0.00s)
    --- PASS: TestDeviceReduceCapacity/multiple_capacities,_multiple_allocations (0.00s)
    --- PASS: TestDeviceReduceCapacity/single_capacity_with_single_topology (0.00s)
    --- PASS: TestDeviceReduceCapacity/single_capacity,_single_topology_type,_multiple_topologies (0.00s)
    --- PASS: TestDeviceReduceCapacity/missing_capacity_topology_for_allocation (0.00s)
    --- PASS: TestDeviceReduceCapacity/single_quantity (0.00s)
    --- PASS: TestDeviceReduceCapacity/multiple_capacities,_one_allocation (0.00s)
    --- PASS: TestDeviceReduceCapacity/missing_capacity_name_for_allocation (0.00s)
    --- PASS: TestDeviceReduceCapacity/single_counter (0.00s)
    --- PASS: TestDeviceReduceCapacity/single_block (0.00s)
=== RUN   TestPoolReduceCapacity
=== RUN   TestPoolReduceCapacity/single_allocation
--- PASS: TestPoolReduceCapacity (0.00s)
    --- PASS: TestPoolReduceCapacity/single_allocation (0.00s)
=== RUN   TestSchedulePodForFoozer
=== RUN   TestSchedulePodForFoozer/single_pod,_container,_cpu/mem,_and_foozer
=== RUN   TestSchedulePodForFoozer/no_foozer_big_enough
--- PASS: TestSchedulePodForFoozer (0.00s)
    --- PASS: TestSchedulePodForFoozer/single_pod,_container,_cpu/mem,_and_foozer (0.00s)
    --- PASS: TestSchedulePodForFoozer/no_foozer_big_enough (0.00s)
=== RUN   TestSchedulePodForBigFoozer
=== RUN   TestSchedulePodForBigFoozer/single_pod,_container,_cpu/mem,_and_foozer
=== RUN   TestSchedulePodForBigFoozer/no_foozer_big_enough
--- PASS: TestSchedulePodForBigFoozer (0.00s)
    --- PASS: TestSchedulePodForBigFoozer/single_pod,_container,_cpu/mem,_and_foozer (0.00s)
    --- PASS: TestSchedulePodForBigFoozer/no_foozer_big_enough (0.00s)
PASS
ok  	github.com/johnbelamaric/k8srm-prototype/pkg/schedule	(cached)
cd cmd/schedule && go build
cd cmd/mock-apiserver && go build
```

## Mock APIServer

This repo includes a crude mock API server that can be loaded with the examples
and used to try out scheduling (WIP). It will spit out some errors but you can
ignore them.

```console
k8srm-prototype$ ./cmd/mock-apiserver/mock-apiserver
W0422 13:20:21.238440 2062725 memorystorage.go:93] type info not known for apiextensions.k8s.io/v1, Kind=CustomResourceDefinition
W0422 13:20:21.238598 2062725 memorystorage.go:93] type info not known for apiregistration.k8s.io/v1, Kind=APIService
W0422 13:20:21.238639 2062725 memorystorage.go:267] type info not known for foozer.example.com/v1alpha1, Kind=FoozerConfig
W0422 13:20:21.238666 2062725 memorystorage.go:267] type info not known for devmgmtproto.k8s.io/v1alpha1, Kind=DeviceDriver
W0422 13:20:21.238685 2062725 memorystorage.go:267] type info not known for devmgmtproto.k8s.io/v1alpha1, Kind=DeviceClass
W0422 13:20:21.238700 2062725 memorystorage.go:267] type info not known for devmgmtproto.k8s.io/v1alpha1, Kind=DeviceClaim
W0422 13:20:21.238712 2062725 memorystorage.go:267] type info not known for devmgmtproto.k8s.io/v1alpha1, Kind=DevicePrivilegedClaim
W0422 13:20:21.238723 2062725 memorystorage.go:267] type info not known for devmgmtproto.k8s.io/v1alpha1, Kind=DevicePool
2024/04/22 13:20:21 addr =  [::]:55441
```

The included `kubeconfig` will access that server. For example:

```console
k8srm-prototype$ kubectl --kubeconfig kubeconfig apply -f testdata/drivers.yaml
devicedriver.devmgmtproto.k8s.io/example.com-foozer created
devicedriver.devmgmtproto.k8s.io/example.com-barzer created
devicedriver.devmgmtproto.k8s.io/sriov-nic created
devicedriver.devmgmtproto.k8s.io/vlan created
k8srm-prototype$ kubectl --kubeconfig kubeconfig get devicedrivers
NAME                 AGE
example.com-foozer   2y112d
example.com-barzer   2y112d
sriov-nic            2y112d
vlan                 2y112d
k8srm-prototype$
```

## `schedule` CLI

This is CLI that represents what the scheduler and/or other controllers will do
in a real system. That is, it will take a pod and a list of nodes and schedule
the pod to the node, taking into account the device claims and writing the
results to the various status fields. This doesn't work right now, it needs to
be updated for the most recent changes.

## Types

Types are divided into "claim" types, which form the UX, "capacity" types which
are populated by drivers, and "allocation types" which are used to capture the
results of scheduling.

Claim types are found in [claim_types.go](pkg/schedule/claim_types.go).

When making a claim for a device (or set of devices), the user may either
specify a device managed by a specific driver, or they may specify an arbitrary
"device type"; for example, "sriov-nic". Individual drivers register with the
control plan and publish the device types which they handle using the
cluster-scoped `DeviceDriver` resource. Examples:
[drivers.yaml](testdata/drivers.yaml).

Vendors and administrators create `DeviceClass` resources to pre-configure
various options for claims. DeviceClass resources must refer to a specific
DeviceType, and may refer to a specific DeviceDriver. Examples:
[classes.yaml](testdata/classes.yaml).

Users create `DeviceClaim` resources, which must refer to a specific
DeviceClass resource. The rest of the DeviceClaim spec can be used to further
specify configuration and selection criteria for the set of desired devices.

DeviceClaim resources are embedded or referenced from the PodSpec, much like
volumes. We should discuss whether we need a separate `DeviceClaimTemplate`
class or if we can simply refer to a DeviceClaim as if it were a temlate.
Probably the separate resource type is cleaner. Examples may be found in the
`testdata` directory in files starting with `pod-`; e.g.,
[pod-template-foozer-single.yaml](testdata/pod-template-foozer-single.yaml).
