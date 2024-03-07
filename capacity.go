package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
)

type NodeResources struct {
	Name string `json:"name"`

	Core     ResourcePool   `json:"core"`
	Extended []ResourcePool `json:"extended,omitempty"`
}

type ResourcePool struct {
	Driver string `json:"driver"`

	// attributes for constraints at the pool level
	Attributes []Attribute `json:"attributes,omitempty"`

	Resources []Resource `json:"resources,omitempty"`
}

type Resource struct {
	Name string `json:"name"`

	// attributes for constraints
	Attributes []Attribute `json:"attributes,omitempty"`

	// topologies for all capacities, unless the capacity
	// overrides it
	Topologies []Topology `json:"topologies,omitempty"`

	// capacities that can be allocated
	Capacities []Capacity `json:"capacities,omitempty"`
}

type Attribute struct {
	Name string `json:"name"`

	// One of the following:
	StringValue   *string            `json:"stringValue,omitempty"`
	IntValue      *int               `json:"intValue,omitempty"`
	QuantityValue *resource.Quantity `json:"quantityValue,omitempty"`
	SemVerValue   *SemVer            `json:"semVerValue,omitempty"`
}

type SemVer string

// This prototype does not address limitations of actuation. We
// would need to do that in the real deal. For example, today
// topology acuation is controlled at the node level, so it is not
// something we can just arbitrarily assign for any node. Instead,
// we need to look at the static topology policy of the node, and evaluate
// if that node assignment can meet the topology constraint in the request
// based upon that policy.
type Topology struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Aggregate bool   `json:"aggregate"` // if true, capacity can be aggregated across this topology
}

type Capacity struct {
	Name string `json:"name"`

	Topologies []Topology `json:"topologies,omitempty"`

	// exactly one of the fields should be populated; that dictates which
	// numerical model to use. Below, two different types of models are
	// shown as examples:
	// a counter model (capacity is decreased in integer increments), and
	// a block counter (capacity is decreased in blocks). How each might
	// be used will be shown below.

	// +optional
	Counter *ResourceCounter `json:"counter,omitempty"`

	// +optional
	Block *ResourceBlock `json:"block,omitempty"`
}

type ResourceCounter struct {
	Capacity int64 `json:"capacity"`
}

type ResourceBlock struct {
	Size     resource.Quantity `json:"size"`
	Capacity resource.Quantity `json:"capacity"`
}

func (c Capacity) AllocateRequest(cr CapacityRequest) (*CapacityRequest, error) {
	if c.Counter != nil && cr.Counter != nil {
		if cr.Counter.Request <= c.Counter.Capacity {
			return &CapacityRequest{
				Capacity: cr.Capacity,
				Counter:  &ResourceCounterRequest{cr.Counter.Request},
			}, nil
		}
		return nil, nil
	}

	if c.Block != nil && cr.Quantity != nil {
		realRequest, ok := cr.Quantity.Request.AsInt64()
		if !ok {
			return nil, fmt.Errorf("could not convert %v to int64", cr.Quantity.Request)
		}
		block, ok := c.Block.Size.AsInt64()
		if !ok {
			return nil, fmt.Errorf("could not convert %v to int64", c.Block.Size)
		}
		remainder := realRequest % block
		if remainder > 0 {
			realRequest = realRequest + block - remainder
		}
		capQuant, ok := c.Block.Capacity.AsInt64()
		if !ok {
			return nil, fmt.Errorf("could not convert %v to int64", c.Block.Capacity)
		}
		if realRequest <= capQuant {
			return &CapacityRequest{
				Capacity: cr.Capacity,
				Quantity: &ResourceQuantityRequest{*resource.NewQuantity(realRequest, "")},
			}, nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("invalid allocation request of %v from %v", cr, c)
}
