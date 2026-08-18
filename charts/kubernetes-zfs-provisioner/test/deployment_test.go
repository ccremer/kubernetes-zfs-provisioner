package test

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"

	"github.com/gruntwork-io/terratest/modules/helm"
	appv1 "k8s.io/api/apps/v1"
)

var tplDeployment = []string{"templates/deployment.yaml"}

func Test_Deployment_ShouldRender_EnvironmentVariables(t *testing.T) {
	options := &helm.Options{
		ValuesFiles: []string{"values/deployment_1.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplDeployment)

	var deployment appv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)

	envs := deployment.Spec.Template.Spec.Containers[0].Env
	require.Contains(t, envs, v1.EnvVar{Name: "KEY1", Value: "value"})
	require.Contains(t, envs, v1.EnvVar{Name: "ANOTHER_KEY", Value: "another value"})
	require.Contains(t, envs, v1.EnvVar{Name: "ZFS_PROVISIONER_INSTANCE", Value: "pv.kubernetes.io/zfs"})
	require.Contains(t, envs, v1.EnvVar{Name: "ZFS_SSH_MOUNT_PATH", Value: "/home/zfs/.ssh"})
}

func Test_Deployment_ShouldRender_SshVolumes_IfEnabled(t *testing.T) {
	options := &helm.Options{
		ValuesFiles: []string{"values/deployment_2.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplDeployment)

	var deployment appv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)

	volumeMounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	require.Contains(t, volumeMounts, v1.VolumeMount{
		Name:      "ssh",
		MountPath: "/home/zfs/.ssh",
		ReadOnly:  true,
	})

	volumes := deployment.Spec.Template.Spec.Volumes
	require.Contains(t, volumes, v1.Volume{
		Name: "ssh",
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  releaseName + "-kubernetes-zfs-provisioner",
				DefaultMode: getIntPointer(0600),
			},
		},
	})
}

func getIntPointer(mode int) *int32 {
	i := *((*int32)(unsafe.Pointer(&mode)))
	return &i
}
