package main

type NodeResources struct {
	Name string `json:"name"`

	Core     ResourcePool   `json:"core"`
	Extended []ResourcePool `json:"extended,omitempty"`
}

type ResourcePool struct {
	Driver string

	// attributes for constraints at the pool
	Attributes []Attribute `json:"attributes,omitempty"`

	Resources []Resource `json:"resources,omitempty"`
}

type Resource struct {
	Name string

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
	StringValue   *string   `json:"stringValue,omitempty"`
	IntValue      *int      `json:"intValue,omitempty"`
	QuantityValue *Quantity `json:"quantityValue,omitempty"`
	SemVerValue   *SemVer   `json:"semVerValue,omitempty"`
}

type Quantity string
type SemVer string

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
	Size     Quantity `json:"size"`
	Capacity Quantity `json:"capacity"`
}
