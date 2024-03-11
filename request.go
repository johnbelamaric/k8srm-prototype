package main

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

type PodCapacityClaim struct {
	// PodClaim contains the resource claims needed for the pod
	// level, such as devices
	PodClaim CapacityClaim

	// ContainerClaims contains the resource claims needed on a
	// per-container level, such as CPU and memory
	ContainerClaims []CapacityClaim

	// Topologies specifies the topological alignment and preferences
	// across all containers and resources in the pod
	Topologies []TopologyConstraint `json:"topologies,omitempty"`
}

type CapacityClaim struct {

	// Claims contains the set of resource claims that are part of
	// this capacity claim
	Claims []ResourceClaim `json:"claims, omitempty"`

	// Topologies specifies the topological alignment and preferences
	// across all resources in this capacity claim
	Topologies []TopologyConstraint `json:"topologies,omitempty"`
}

type ResourceClaim struct {
	// Driver will limit the scope of resources considered
	// to only those published by the specified driver
	Driver string `json:"driver,omitempty"`

	// Constraints is a CEL expression that operates on
	// node and resource attributes, and must evaluate to true
	// for a resource to be considered
	Constraints string `json:"constraints,omitempty"`

	// Topologies specifies topological alignment constraints and
	// preferences for the allocated capacities. These constraints
	// apply across the capacities within the resource.
	Topologies []TopologyConstraint `json:"topologies,omitempty"`

	// Capacities specifies the individual allocations needed
	// from the capacities provided by the resource
	Capacities []CapacityRequest `json:"capacities,omitempty"`
}

type TopologyConstraint struct {
	// Type identifies the type of topology to constrain
	Type string `json:"type"`

	// Policy defines the specific constraint. All types support 'prefer'
	// and 'require', with 'prefer' being the default. 'Prefer' means
	// that allocations will be made according to the topology when
	// possible, but the allocation will not fail if the constraint cannot
	// be met. 'Require' will fail the allocation if the constraint is not
	// met. Types may add additional policies.
	Policy string `json:"policy,omitempty"`
}

type CapacityRequest struct {
	Capacity string `json:"capacity"`

	// one of these must be populated
	// note that we only need two different type of capacity requests
	// even though we have three different types of capacity models
	// the ResourceQuantity and ResourceBlock capacity models both
	// are drawn down on via the ResourceQuantityRequest type
	Counter  *ResourceCounterRequest  `json:"counter,omitempty"`
	Quantity *ResourceQuantityRequest `json:"quantity,omitempty"`
}

type ResourceCounterRequest struct {
	Request int64 `json:"request"`
}

type ResourceQuantityRequest struct {
	Request resource.Quantity `json:"request"`
}
