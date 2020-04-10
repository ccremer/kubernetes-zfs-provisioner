package provisioner

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewStorageClassParameters(t *testing.T) {
	type args struct {
		parameters map[string]string
	}
	tests := []struct {
		name        string
		args        args
		want        *ZFSStorageClassParameters
		errContains string
	}{
		{
			name: "GivenWrongSpec_WhenParentDatasetEmpty_ThenThrowError",
			args: args{
				parameters: map[string]string{
					hostnameParameter: "host",
				},
			},
			errContains: parentDatasetParameter,
		},
		{
			name: "GivenWrongSpec_WhenParentDatasetBeginsWithSlash_ThenThrowError",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter: "/tank",
					hostnameParameter:      "host",
					typeParameter:          "nfs",
				},
			},
			errContains: parentDatasetParameter,
		},
		{
			name: "GivenWrongSpec_WhenParentDatasetEndsWithSlash_ThenThrowError",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter: "/tank/volume/",
					hostnameParameter:      "host",
					typeParameter:          "nfs",
				},
			},
			errContains: parentDatasetParameter,
		},
		{
			name: "GivenWrongSpec_WhenHostnameEmpty_ThenThrowError",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter: "tank",
				},
			},
			errContains: hostnameParameter,
		},
		{
			name: "GivenWrongSpec_WhenTypeInvalid_ThenThrowError",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter: "tank",
					hostnameParameter:      "host",
					typeParameter:          "invalid",
				},
			},
			errContains: typeParameter,
		},
		{
			name: "GivenCorrectSpec_WhenTypeNfs_ThenReturnNfsParameters",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter:   "tank",
					hostnameParameter:        "host",
					typeParameter:            "nfs",
					sharePropertiesParameter: "rw",
				},
			},
			want: &ZFSStorageClassParameters{NFS: &NFSParameters{ShareProperties: "rw"}},
		},
		{
			name: "GivenCorrectSpec_WhenTypeNfsWithoutProperties_ThenReturnNfsParametersWithDefault",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter: "tank",
					hostnameParameter:      "host",
					typeParameter:          "nfs",
				},
			},
			want: &ZFSStorageClassParameters{NFS: &NFSParameters{ShareProperties: "on"}},
		},
		{
			name: "GivenCorrectSpec_WhenTypeHostPath_ThenReturnHostPathParameters",
			args: args{
				parameters: map[string]string{
					parentDatasetParameter: "tank",
					hostnameParameter:      "host",
					typeParameter:          "hostpath",
					nodeNameParameter:      "my-node",
				},
			},
			want: &ZFSStorageClassParameters{HostPath: &HostPathParameters{NodeName: "my-node"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewStorageClassParameters(tt.args.parameters)
			if tt.errContains != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want.NFS, result.NFS)
			assert.Equal(t, tt.want.HostPath, result.HostPath)
		})
	}
}
