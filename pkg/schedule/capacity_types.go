package schedule

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DevMgmtAPIVersion = "devmgmtproto.k8s.io/v1alpha1"
)

// This prototype models nodes as a collection of device
// pools, each populated by devices, which in turn hold
// capacities.
//

// NOTE: probably obsolete, leaving for now
type NodeDevices struct {
	Name string `json:"name"`

	Pools []DevicePool `json:"pools"`
}

// DevicePool represents a collection of devices managed by a given driver. How
// devices are divided into pools is driver-specific, but typically the
// expectation would a be a pool per identical collection of devices, per node.
//

type DevicePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevicePoolSpec   `json:"spec,omitempty"`
	Status DevicePoolStatus `json:"status,omitempty"`
}

// DevicePoolSpec identifies the driver and contains the details of all devices prior to any allocations.
// NOTE: It's not clear that spec/status is the right model for this data.
type DevicePoolSpec struct {

	// Driver is the name of the DeviceDriver that created this object and
	// owns the data in it.
	// +required
	Driver string `json:"driver"`

	// Attributes contains device attributes that are common to all devices
	// in the pool.
	// +optional
	Attributes []Attribute `json:"attributes,omitempty"`

	// DeviceCount contains the total number of devices in the pool.
	// +required
	DeviceCount int `json:"count,omitempty"`

	// Devices contains the individual devices in the pool. This is only
	// needed if devices have additional attributes beyond what the pool
	// already identified, or if the driver supports per-device resources.
	// If used, len(Devices) must equal DeviceCount.
	// +optional
	Devices []Device `json:"devices,omitempty"`
}

// DevicePoolStatus contains the state of the pool as last reported by the
// driver. Note that this will not include the allocations that have been made
// by the scheduler but not yet seen by the driver. Thus, it is NOT sufficient
// to make future scheduling decisions.
type DevicePoolStatus struct {
	AvailableDevices int `json:"availableDevices,omitempty"`
}

// Device is used to track individual devices in a pool, for drivers that
// support per-device attributes or resources, or need to otherwise specify
// specific devices that satisfy a claim.
type Device struct {
	Name string `json:"name"`

	// attributes for constraints
	Attributes []Attribute `json:"attributes,omitempty"`

	// resources that can be allocated
	Resources []ResourceCapacity `json:"capacities,omitempty"`
}

type ResourceCapacity struct {
	Name string `json:"name"`

	Topologies []Topology `json:"topologies,omitempty"`

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
	Name string `json:"name"`
	Type string `json:"type"`

	// GroupInDevice allows a claim to be satisfied by capacities from
	// different topologies, but in the same device.
	GroupInDevice bool `json:"groupInDevice"`
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
