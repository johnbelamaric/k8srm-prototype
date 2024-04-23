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

// For clarify, types relating to supporting individual devices and per-device
// resources are found in device_types.go.
