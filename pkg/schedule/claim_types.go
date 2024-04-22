package schedule

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// This prototype models requests for capacity as DeviceClaims, which
// are structured based on the workload structure. For example, we can
// request capacity required to run a pod. This includes device claims
// for the pod itself (for example, we have a counter for number of pods
// allowed on a node), as well as device claims for each container in
// the pod. Claims may include CEL-based constraints, as well as topological
// constraints. Those topological constraints may apply to the whole pod
// (equivalent to topology manager scope=pod), or to individual containers
// in the pod (equivalent to topology manager scope=container).
//
//

// DeviceClass is a vendor or admin-provided resource that contains
// contraint and configuration information. Essentially, it is a re-usable
// collection of predefined data that device claims may use.
// Cluster scoped.
type DeviceClass struct {
	// Name is used to identify the device class.
	// +required
	Name string `json:"name"`

	// DeviceType is a driver-independent classification of the device.
	// Alternatively, we may want to consider a DeviceCapabilities vector,
	// or use device attributes or individual resource types supplied by a
	// device to indicate device functions.
	// +optional
	DeviceType string `json:deviceType,omitempty`

	// Driver specifies the driver that should handle this class of devices.
	// When a DeviceClaim uses this class, only devices published by the
	// specified driver will be considered.
	// +optional
	Driver string `json:driver,omitempty`

	// Constraints is a CEL expression that operates on device attributes,
	// and must evaluate to true for a device to be considered. It will be
	// ANDed with any Constraints field in the DeviceClaim using this class.
	// +optional
	Constraints string `json:"constraints,omitempty"`

	// Device classes and claims may represent or be satisfied by choosing
	// multiple devices instead of just a single device.

	// MinDeviceCount is the minimum number of devices that should be selected
	// when satsifying a claim using this class. Default is 1.
	// +optional
	MinDeviceCount int `json:"minDeviceCount,omitempty"`

	// MaxDeviceCount is the maximum number of devices that should be selected
	// when sastisfying a claim using this class. No maximum, by default.
	// +optional
	MaxDeviceCount int `json:"maxDeviceCount,omitempty"`

	// AccessMode defines whether device claims using this class are requesting
	// exclusive access or can allow shared access. If not specified, then the
	// value from the claim will be used. Otherwise, the class value takes
	// precedence (or better yet, we flag an error).
	// +optional
	AccessMode DeviceAccessMode `json:"accessMode,omitempty"`

	// DeviceConfigs contains references to arbitrary vendor device configuration
	// objects that will be attached to the device allocation.
	// +optional
	Configs []DeviceClassConfigReference `json:"configs,omitempty"`
}

// DeviceClaim is used to specify a request for a set of devices.
type DeviceClaim struct {
	metav1.TypeMeta   `"json:,inline"`
	metav1.ObjectMeta `"json:metadata,omitempty"`

	Spec   DeviceClaimSpec   `"json:spec,omitempty"`
	Status DeviceClaimStatus `"json:status,omitempty"`
}

// DeviceClaimSpec details the requirements that devices chosen
// to satisfy this claim must meet.
type DeviceClaimSpec struct {
	// DeviceClass is the name of the DeviceClass containing the basic information
	// about the device being requested.
	// +required
	DeviceClass string `json:"deviceClass"`

	// Driver will limit the scope of devices considered to only those
	// published by the specified driver. If the DeviceClass specifies a
	// Driver, this should be left empty. If it is not, then it MUST match
	// the Driver in the DeviceClass
	// +optional
	Driver string `json:"driver,omitempty"`

	// Constraints is a CEL expression that operates on device attributes.
	// In order for a device to be considered, this CEL expression and the
	// Constraints expression from the DeviceClass must both be true.
	// +optional
	Constraints string `json:"constraints,omitempty"`

	// Device classes and claims may represent or be satisfied by choosing
	// multiple devices instead of just a single device.

	// MinDeviceCount is the minimum number of devices that should be selected
	// for this claim. It must be greater than or equal to the calss MinDeviceCount,
	// and less than or equal to the class MaxDeviceCount. Default is 1.
	// +optional
	MinDeviceCount int `json:"minDeviceCount,omitempty"`

	// MaxDeviceCount is the maximum number of devices that should be selected
	// for this claim. It must be less than or equal to the class MaxDeviceCount.
	// Default is no maximum.
	// +optional
	MaxDeviceCount int `json:"maxDeviceCount,omitempty"`

	// AccessMode defines whether this claim requires exclusive access or can
	// allow shared access. If not specified, then the value from the class
	// will be used. If neither is specified, exclusive access is the default.
	// +optional
	AccessMode DeviceAccessMode `json:"accessMode,omitempty"`

	// Topologies specifies topological alignment constraints and
	// preferences for the allocated resources. These constraints
	// apply across the resources within the set of devices.
	// +optional
	Topologies []TopologyConstraint `json:"topologies,omitempty"`

	// Requests specifies the individual allocations needed
	// from the capacities provided by the device
	// +optional
	Requests []CapacityRequest `json:"requests,omitempty"`

	// Configs contains references to arbitrary vendor device configuration
	// objects that will be attached to the device allocation.
	// +optional
	Configs []DeviceConfigReference `json:"configs,omitempty"`
}

type DeviceClaimStatus struct {
	// ClassConfigs contains the entire set of dereferenced vendor
	// configurations from the DeviceClass, as of the time of allocation.
	// +optional
	ClassConfigs []runtime.RawExtension

	// ClaimConfigs contains the entire set of dereferenced vendor
	// configurations from the DeviceClaim, as of the time of allocation.
	// +optional
	ClaimConfigs []runtime.RawExtension

	// Allocation contains the selected devices, along with
	// their resource allocations and topology assignment.
	Allocation DeviceResult `json:"allocation,omitempty"`

	// PodNames contains the names of all Pods using this claim.
	// TODO: Can we just use ownerRefs instead?
	PodNames []string
}

type DeviceClassConfigReference struct {
	// API version of the referent.
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind of the referent.
	// +required
	Kind string `json:"kind"`

	// Namespace of the referent.
	// +required
	Namespace string `json:"namespace"`

	// Name of the referent.
	// +required
	Name string `json:"name"`
}

type DeviceConfigReference struct {
	// API version of the referent.
	// +required
	APIVersion string `json:"apiVersion"`

	// Kind of the referent.
	// +required
	Kind string `json:"kind"`

	// Name of the referent.
	// +required
	Name string `json:"name"`
}

type DeviceAccessMode string

type PodCapacityClaim struct {
	// PodClaim contains the device claims needed for the pod
	// level, such as the pod capacity needed to run pods,
	// or devices that may be attached to a container later
	// +required
	PodClaim CapacityClaim `json:"podClaim"`

	// ContainerClaims contains the device claims needed on a
	// per-container level, such as CPU and memory
	// +required
	ContainerClaims []CapacityClaim `json:"containerClaims"`

	// Topologies specifies the topological alignment and preferences
	// across all containers and devices in the pod
	// +optional
	Topologies []TopologyConstraint `json:"topologies,omitempty"`
}

type CapacityClaim struct {
	// Name is used to identify the capacity claim to help in troubleshooting
	// unschedulable claims.
	// +required
	Name string `json:"name"`

	// Claims contains the set of device claims that are part of
	// this capacity claim
	// +required
	Claims []DeviceClaim `json:"claims"`

	// Topologies specifies the topological alignment and preferences
	// across all devices in this capacity claim
	// +optional
	Topologies []TopologyConstraint `json:"topologies,omitempty"`
}

type TopologyConstraint struct {
	// Type identifies the type of topology to constrain
	// +required
	Type string `json:"type"`

	// Policy defines the specific constraint. All types support 'prefer'
	// and 'require', with 'prefer' being the default. 'Prefer' means
	// that allocations will be made according to the topology when
	// possible, but the allocation will not fail if the constraint cannot
	// be met. 'Require' will fail the allocation if the constraint is not
	// met. Types may add additional policies.
	// +optional
	Policy string `json:"policy,omitempty"`
}

type CapacityRequest struct {
	// +required
	Capacity string `json:"capacity"`

	// one of these must be populated
	// note that we only need three different type of capacity requests
	// even though we have four different types of capacity models
	// the ResourceQuantity and ResourceBlock capacity models both
	// are drawn down on via the ResourceQuantityRequest type
	Counter    *ResourceCounterRequest    `json:"counter,omitempty"`
	Quantity   *ResourceQuantityRequest   `json:"quantity,omitempty"`
	AccessMode *ResourceAccessModeRequest `json:"accessMode,omitempty"`
}

type ResourceCounterRequest struct {
	// +required
	Request int64 `json:"request"`
}

type ResourceQuantityRequest struct {
	// +required
	Request resource.Quantity `json:"request"`
}

type CapacityAccessMode string

const (
	ReadOnlyShared     = "ReadOnlyShared"
	ReadWriteShared    = "ReadWriteShared"
	WriteExclusive     = "WriteExclusive"
	ReadWriteExclusive = "ReadWriteExclusive"
)

type ResourceAccessModeRequest struct {
	// +required
	Request CapacityAccessMode `json:"request"`
}
