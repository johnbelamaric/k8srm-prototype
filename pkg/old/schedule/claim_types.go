package schedule

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeviceDriver is a vendor provided resource that registers a given
// driver with the cluster.
// Cluster scoped.
type DeviceDriver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// DeviceTypes specifies which DeviceType values are handled by this
	// driver. DeviceType is a driver-independent classification of the
	// device. In particular, for well-understood standards like SR-IOV
	// based network interfaces, a device claim should be satisfiable by
	// any vendor's devices, subject to the CEL-based Constraints fields in
	// the class and claim.
	//
	// Drivers must register which device types they support. The code
	// itself need not understand the meaning of the device types; rather,
	// they are just used to map to a set of drivers that may satisfy a
	// claim.
	//
	// +required
	DeviceTypes []string
}

// DeviceClass is a vendor or admin-provided resource that contains
// contraint and configuration information. Essentially, it is a re-usable
// collection of predefined data that device claims may use.
// Cluster scoped.
type DeviceClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceClassSpec   `json:"spec,omitempty"`
	Status DeviceClassStatus `json:"status,omitempty"`
}

// DeviceClassSpec provides the details of the DeviceClass.
type DeviceClassSpec struct {
	// DeviceType is a driver-independent classification of the device.
	// This may be used instead of specifying the Driver explicitly, so that
	// we do not aribtrarily limit claims to a particular vendor's devices.
	//
	// Alternatively, we may want to consider a DeviceCapabilities vector,
	// or use device attributes or individual resource types supported by a
	// device to indicate device functions.
	//
	// +required
	DeviceType string `json:deviceType,omitempty`

	// Driver specifies the driver that should handle this class of devices.
	// When a DeviceClaim uses this class, only devices published by the
	// specified driver will be considered.
	// +optional
	Driver *string `json:driver,omitempty`

	// Constraints is a CEL expression that operates on device attributes,
	// and must evaluate to true for a device to be considered. It will be
	// ANDed with any Constraints field in the DeviceClaim using this class.
	// +optional
	Constraints *string `json:"constraints,omitempty"`

	// Device claims may represent be satisfied by choosing multiple
	// devices instead of just a single device. The min/max fields control
	// whether we want a single device, or a set of devices to satisfy a
	// claim.

	// MinDeviceCount is the minimum number of devices that should be selected
	// when satsifying a claim using this class. Default is 1.
	// +optional
	MinDeviceCount *int `json:"minDeviceCount,omitempty"`

	// MaxDeviceCount is the maximum number of devices that should be selected
	// when sastisfying a claim using this class. No maximum, by default.
	// +optional
	MaxDeviceCount *int `json:"maxDeviceCount,omitempty"`

	// AttributeMatches allows specifying a constraint within a set of chosen
	// devices, without having to explicitly specify the value of the constraint.
	// For example, this allows constraints like "all devices must be the same model",
	// without having to specify the exact model. We may be able to use this for some
	// basic topology constraints too, by representing the topology as device attributes.
	// +optional
	AttributeMatches []string `json:"attributeMatches,omitempty"`

	// AccessMode defines whether device claims using this class are requesting
	// exclusive access or can allow shared access. If not specified, then the
	// value from the claim will be used. Otherwise, the class value takes
	// precedence (or better yet, we flag an error).
	// +optional
	AccessMode *DeviceAccessMode `json:"accessMode,omitempty"`

	// DeviceConfigs contains references to arbitrary vendor device configuration
	// objects that will be attached to the device allocation.
	// +optional
	Configs []DeviceClassConfigReference `json:"configs,omitempty"`
}

// DeviceClassStatus contains the current status of the class in the cluster.
type DeviceClassStatus struct {
	// Conditions contains the latest observation of the class's state.
	// A class will be in Ready state if at least one DeviceDriver is
	// registered to handle the class.
	Conditions []metav1.Condition `json:"conditions"`

	// Drivers contains the list of drivers that can handle this class.
	Drivers []string `json:"drivers,omitempty"`
}

// DeviceClaim is used to specify a request for a set of devices.
// Namespace scoped.
type DeviceClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DeviceClaimSpec   `json:"spec,omitempty"`
	Status DeviceClaimStatus `json:"status,omitempty"`
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
	// the Driver in the DeviceClass.
	// +optional
	Driver *string `json:"driver,omitempty"`

	// Constraints is a CEL expression that operates on device attributes.
	// In order for a device to be considered, this CEL expression and the
	// Constraints expression from the DeviceClass must both be true.
	// +optional
	Constraints *string `json:"constraints,omitempty"`

	// Device classes and claims may represent or be satisfied by choosing
	// multiple devices instead of just a single device.

	// MinDeviceCount is the minimum number of devices that should be selected
	// for this claim. It must be greater than or equal to the calss MinDeviceCount,
	// and less than or equal to the class MaxDeviceCount. Default is 1.
	// +optional
	MinDeviceCount *int `json:"minDeviceCount,omitempty"`

	// MaxDeviceCount is the maximum number of devices that should be selected
	// for this claim. It must be less than or equal to the class MaxDeviceCount.
	// Default is no maximum.
	// +optional
	MaxDeviceCount *int `json:"maxDeviceCount,omitempty"`

	// AttributeMatches allows specifying a constraint within a set of chosen
	// devices. The list here will be merged with the list (if any)  provided
	// in the class.
	// +optional
	AttributeMatches []string `json:"attributeMatches,omitempty"`

	// AccessMode defines whether this claim requires exclusive access or can
	// allow shared access. If not specified, then the value from the class
	// will be used. If neither is specified, exclusive access is the default.
	// +optional
	AccessMode *DeviceAccessMode `json:"accessMode,omitempty"`

	// Configs contains references to arbitrary vendor device configuration
	// objects that will be attached to the device allocation.
	// +optional
	Configs []DeviceConfigReference `json:"configs,omitempty"`

	// NOTE: Topologies and Requests are here for now because they were
	// part of the original prototype. However, we probably won't do
	// anything with topology in 1.31.  Requests are used for per-device
	// resources, which may or may not be needed in 1.31. So, these remain
	// here, but we may want to see how far we can get without them.

	// Topologies specifies topological alignment constraints and
	// preferences for the allocated resources. These constraints
	// apply across the resources within the set of devices.
	// +optional
	Topologies []TopologyConstraint `json:"topologies,omitempty"`

	// Requests specifies the individual allocations needed
	// from the resources provided by the device.
	// +optional
	Requests []CapacityRequest `json:"requests,omitempty"`
}

// DeviceClaimStatus contains the results of the claim allocation.
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
	// +optional
	Allocation *DeviceResult `json:"allocation,omitempty"`

	// PodNames contains the names of all Pods using this claim.
	// TODO: Can we just use ownerRefs instead?
	// +optional
	PodNames []string
}

// DevicePrivilegedClaim is used to specify a special kind of privileged claim
// for a set of devices on a node. This type of claim is used for monitoring or
// other management services for a device. It ignores all ordinary claims to
// the device with respect to access modes and any resource allocations. As a
// separate type, it can (and is expected to) have separate RBAC constraints.
//
// It does not have all the sophisticated selection mechanisms of an ordinary
// device claim, as the most common use case is simply to access all devices
// managed by a given driver on a given node. It intentionally does not require
// a class, though it does allow some flexibility with the specification of
// Constraints and Configs.

type DevicePrivilegedClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DevicePrivilegedClaimSpec   `json:"spec,omitempty"`
	Status DevicePrivilegedClaimStatus `json:"status,omitempty"`
}

// DevicePrivilegedClaimSpec contains the details of the privileged claim.
type DevicePrivilegedClaimSpec struct {
	// Driver will limit the scope of devices considered to only those
	// published by the specified driver.
	// +required
	Driver string `json:"driver,omitempty"`

	// Constraints is a CEL expression that operates on device attributes.
	// Only devices matching this constraint will be selected by this
	// claim.
	// +optional
	Constraints *string `json:"constraints,omitempty"`

	// Configs contains references to arbitrary vendor device configuration
	// objects that will be attached to the device allocation.
	// +optional
	Configs []DeviceConfigReference `json:"configs,omitempty"`
}

// DevicePrivilegedClaimStatus contains the results of the claim allocation.
type DevicePrivilegedClaimStatus struct {
	// ClaimConfigs contains the entire set of dereferenced vendor
	// configurations from the DeviceClaim, as of the time of allocation.
	// +optional
	ClaimConfigs []runtime.RawExtension

	// Allocation contains the selected devices, along with
	// their resource allocations and topology assignment.
	// +optional
	Allocation *DeviceResult `json:"allocation,omitempty"`

	// PodNames contains the names of all Pods using this claim.
	// TODO: Can we just use ownerRefs instead?
	// +optional
	PodNames []string
}

// DeviceClassConfigReference is used to refer to arbitrary configuration
// objects from the class. Since it is the class, and therefore is created by
// the administrator, it allows referencing objects in any namespace.
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

// DeviceConfigReference is used to refer to arbitrary configuration object
// from the claim. Since it is created by the end user, the referenced objects
// are restricted to the same namespace as the DeviceClaim.
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

// DeviceAccessMode represents access modes for a device, such as shared or
// exclusive mode.
type DeviceAccessMode string

// NOTE: The types below are internal and may be phased out soon. They are from
// the prior version of the prototype and I haven't decided what to do with
// them yet.

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
	Resource string `json:"resource"`

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