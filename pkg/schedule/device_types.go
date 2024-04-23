package schedule

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// This file contains the types relevant to drivers that publish
// per-device information.

// Claim spec types. This section contains types used in the claim spec to
// make requests for specific device resources and access.

// DeviceAccessMode represents access modes for a device, such as shared or
// exclusive mode.
type DeviceAccessMode string

type CapacityRequest struct {
	// Resource contains the resource type/name.
	// +required
	Resource string `json:"resource"`

	// The fields below represent the actual quantity of the request.
	// Exactly one must be populated.  Note that we only need three
	// different type of capacity requests even though we have four
	// different types of capacity models the ResourceQuantity and
	// ResourceBlock capacity models both are drawn down on via the
	// ResourceQuantityRequest type.
	//
	// Access mode is currently modeled like another resource, but this
	// will probably be changed.
	Counter    *ResourceCounterRequest    `json:"counter,omitempty"`
	Quantity   *ResourceQuantityRequest   `json:"quantity,omitempty"`
	AccessMode *ResourceAccessModeRequest `json:"accessMode,omitempty"`
}

type ResourceCounterRequest struct {
	// Request contains the number of individual resources needed.
	// +required
	Request int64 `json:"request"`
}

type ResourceQuantityRequest struct {
	// Request contains the quantity of resources needed.
	// +required
	Request resource.Quantity `json:"request"`
}

const (
	ReadOnlyShared     = "ReadOnlyShared"
	ReadWriteShared    = "ReadWriteShared"
	WriteExclusive     = "WriteExclusive"
	ReadWriteExclusive = "ReadWriteExclusive"
)

type ResourceAccessModeRequest struct {
	// Request contains the desired access mode.
	// +required
	Request DeviceAccessMode `json:"request"`
}

// Claim status types. This section contains the types for storing the results
// of an allocation using individual devices and per-device resources.

// DeviceAllocation contains an individual device allocation result, including
// per-device resource allocations, when applicable.
type DeviceAllocation struct {
	// DeviceName contains the name of the allocated Device.
	// +required
	DeviceName string `json:"deviceName,omitempty"`

	// Allocations contain the per-device resource allocations for
	// this device.
	// +optional
	Allocations []ResourceAllocation `json:"allocations,omitempty"`
}

// ResourceAllocation contains the per-device resource allocations.
type ResourceAllocation struct {
	// CapacityRequest contains the allocated amount (which may
	// be different than the original, requested amount.
	CapacityRequest `json:",inline"`

	// When we support topology, it is likely we will need to add
	// topology assignments in here.
}

// Capacity types. This section contains the types needed to publish the
// individual devices available and their per-device resources.

// Device is used to track individual devices in a pool, for drivers that
// support per-device attributes or resources, or need to otherwise specify
// specific devices that satisfy a claim. NOTE: We may be able to change
// Attributes here to just some topology information, and normalize Resources
// up into the pool. That is, the pool would provide a basic shape of the
// device, and the device list would just provide individual device names. When
// we make allocations, we would track what allocations are made by device
// name, and so apply those allocations to the base resource values prior to
// each scheduling attempt (this can be cached).
type Device struct {
	// Name is a driver-specific identifier for the device.
	// +required
	Name string `json:"name"`

	// Attributes contain additional metadata that can be used in
	// constraints. If an attribute name overlaps with the pool attribute,
	// the device attribute takes precedence.
	// +optional
	Attributes []Attribute `json:"attributes,omitempty"`

	// Resources allows the definition of per-device resources that can
	// be allocated in a manner similar to standard Kubernetes resources.
	// +optional
	Resources []ResourceCapacity `json:"resources,omitempty"`
}

type ResourceCapacity struct {
	Name string `json:"name"`

	// exactly one of the fields should be populated
	// examples implemented:
	//  - counter: integer capacity decremented by integers
	//  - quantity: resource.Quantity capacity decremented by quantities
	//  - block:  resource.Quantity capacity decremented in blocks
	//  - accessMode: allows various controlled access:
	//       - readonly-shared: allowed with other consumers using *-shared, write-exclusive
	//       - readwrite-shared: allowed with other consumers using *-shared
	//       - write-exclusive: allowed other consumers using readonly-shared
	//       - readwrite-exclusive: no other consumers allowed

	// +optional
	Counter *ResourceCounter `json:"counter,omitempty"`

	// +optional
	Quantity *ResourceQuantity `json:"quantity,omitempty"`

	// +optional
	Block *ResourceBlock `json:"block,omitempty"`

	// +optional
	AccessMode *ResourceAccessMode `json:"accessMode,omitempty"`
}

type ResourceCounter struct {
	Capacity int64 `json:"capacity"`
}

type ResourceQuantity struct {
	Capacity resource.Quantity `json:"capacity"`
}

type ResourceBlock struct {
	Size     resource.Quantity `json:"size"`
	Capacity resource.Quantity `json:"capacity"`
}

type ResourceAccessMode struct {
	// if not allowed, any requests for that access mode
	// will be converted to a request for the next highest
	// allowed access mode.
	AllowReadOnlyShared  bool `json:"allowReadOnlyShared"`
	AllowReadWriteShared bool `json:"allowReadWriteShared"`
	AllowWriteExclusive  bool `json:"allowWriteExclusive"`

	// tracks reference counts for each access mode
	ReadOnlyShared     int `json:"readOnlyShared"`
	ReadWriteShared    int `json:"readWriteShared"`
	WriteExclusive     int `json:"writeExclusive"`
	ReadWriteExclusive int `json:"readWriteExclusive"`
}
