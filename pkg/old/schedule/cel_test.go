package schedule

import (
	"testing"

	"github.com/stretchr/testify/require"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestMeetsConstraints(t *testing.T) {
	testCases := map[string]struct {
		device      Device
		constraints *string
		poolAttrs   []Attribute
		expErr      string
		result      bool
	}{
		"nil constraint": {
			device:      Device{},
			constraints: nil,
			result:      true,
		},
		"empty constraint": {
			device:      Device{},
			constraints: ptr(""),
			result:      true,
		},
		"simple device constraint met": {
			device: Device{
				Attributes: []Attribute{
					{
						Name:        "vendor",
						StringValue: ptr("example.com"),
					},
				},
			},
			constraints: ptr("device.vendor == 'example.com'"),
			result:      true,
		},
		"simple device constraint failed": {
			device: Device{
				Attributes: []Attribute{
					{
						Name:        "vendor",
						StringValue: ptr("example.org"),
					},
				},
			},
			constraints: ptr("device.vendor == 'example.com'"),
			result:      false,
		},
		"simple device and pool constraint met": {
			device: Device{
				Attributes: []Attribute{
					{
						Name:        "model",
						StringValue: ptr("foozer-1000"),
					},
				},
			},
			poolAttrs: []Attribute{
				{
					Name:        "vendor",
					StringValue: ptr("example.com"),
				},
			},
			constraints: ptr("device.vendor == 'example.com' && device.model == 'foozer-1000'"),
			result:      true,
		},
		"simple device and pool constraint failed": {
			device: Device{
				Attributes: []Attribute{
					{
						Name:        "model",
						StringValue: ptr("foozer-1000"),
					},
				},
			},
			poolAttrs: []Attribute{
				{
					Name:        "vendor",
					StringValue: ptr("example.org"),
				},
			},
			constraints: ptr("device.vendor == 'example.com' && device.model == 'foozer-1000'"),
			result:      false,
		},
		//TODO: add CEL type conversion for resource.Quantity so this test below can be enabled
		"quantity constraint met": {
			device: Device{
				Attributes: []Attribute{
					{
						Name:          "memory",
						QuantityValue: ptr(resource.MustParse("10Gi")),
					},
				},
			},
			expErr:      "unsupported conversion to ref.Val: (resource.Quantity){{10737418240 0} {<nil>} 10Gi BinarySI}",
			constraints: ptr("device.memory >= '10Gi'"),
			result:      true,
		},
	}
	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			result, err := tc.device.MeetsConstraints(tc.constraints, tc.poolAttrs)
			if tc.expErr == "" {
				require.NoError(t, err)
				require.Equal(t, tc.result, result)
			} else {
				require.EqualError(t, err, tc.expErr)
			}
		})
	}
}
