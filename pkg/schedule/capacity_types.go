package schedule

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DevMgmtAPIVersion = "devmgmtproto.k8s.io/v1alpha1"
)

// DevicePool represents a collection of devices managed by a given driver. How
// devices are divided into pools is driver-specific, but typically the
// expectation would a be a pool per identical collection of devices, per node.
type DevicePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevicePoolSpec   `json:"spec,omitempty"`
	Status DevicePoolStatus `json:"status,omitempty"`
}

// DevicePoolSpec identifies the driver and contains the details of all devices prior to any allocations.
// NOTE: It's not clear that spec/status is the right model for this data.
type DevicePoolSpec struct {
	// NodeName is the name of the node containing the devices in the pool.
	// For network attached devices, this may be empty.
	// +optional
	NodeName *string `json:"nodeName,omitempty"`

	// Driver is the name of the DeviceDriver that created this object and
	// owns the data in it.
	// +required
	Driver string `json:"driver,omitempty"`

	// Attributes contains device attributes that are common to all devices
	// in the pool.
	// +optional
	Attributes []Attribute `json:"attributes,omitempty"`

	// DeviceCount contains the total number of devices in the pool.
	// +required
	DeviceCount int `json:"count,omitempty"`

	// Devices contains the individual devices in the pool. Some features
	// require tracking specific devices, in which case this should be
	// populated. Populating individual devices is required for these
	// features:
	// - Access modes (shared vs exclusive). Drivers can implement some
	//   sharing between pods without listing individual devices, if the
	//   drivers themselves maintain a local mapping of claim to devices.
	// - Non-homogenous devices in a pool
	// - Per-device resources
	//
	// If used, len(Devices) must equal DeviceCount.
	//
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

// Attribute capture the name, value, and type of an device attribute.
type Attribute struct {
	Name string `json:"name"`

	// One of the following:
	StringValue   *string            `json:"stringValue,omitempty"`
	IntValue      *int               `json:"intValue,omitempty"`
	QuantityValue *resource.Quantity `json:"quantityValue,omitempty"`
	SemVerValue   *SemVer            `json:"semVerValue,omitempty"`
}

// SemVer represents a semantic version value. In this prototype it is just a
// string.
type SemVer string

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
