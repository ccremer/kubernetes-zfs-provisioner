package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var tplSecret = []string{"templates/secret.yaml"}

func Test_Secret_GivenNoExternalSecret_WhenConfigSet_ThenRenderConfigFile(t *testing.T) {
	options := &helm.Options{
		ValuesFiles: []string{"values/secret_1.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplSecret)

	var secret v1.Secret
	helm.UnmarshalK8SYaml(t, output, &secret)

	config := secret.StringData["config"]
	assert.Equal(t, `Host test
  IdentityFile ~/.ssh/id_ed25519`, config)
}

func Test_Secret_GivenNoExternalSecret_WhenIdentitiesSet_ThenRenderPrivateKeys(t *testing.T) {
	options := &helm.Options{
		ValuesFiles: []string{"values/secret_2.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplSecret)

	var secret v1.Secret
	helm.UnmarshalK8SYaml(t, output, &secret)

	expected := "----\nPRIVATE_KEY\n----"

	assert.Contains(t, secret.StringData["id_rsa"], expected)
	assert.Contains(t, secret.StringData["id_ed25519"], expected)
}

func Test_Secret_GivenNoExternalSecret_WhenKnownHostsSet_ThenRenderHostKeys(t *testing.T) {
	options := &helm.Options{
		ValuesFiles: []string{"values/secret_3.yaml"},
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, tplSecret)

	var secret v1.Secret
	helm.UnmarshalK8SYaml(t, output, &secret)

	expectedHost := "test"
	expectedPubKey := "ssh-rsa asdf"

	assert.Contains(t, secret.StringData["known_hosts"], expectedHost+" "+expectedPubKey)
}
