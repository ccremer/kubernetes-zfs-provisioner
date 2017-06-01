package main

import (
	"os/exec"
	"strings"
	"time"

	"git.gentics.com/psc/kubernetes-zfs-provisioner/pkg/provisioner"
	log "github.com/Sirupsen/logrus"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	leasePeriod   = controller.DefaultLeaseDuration
	retryPeriod   = controller.DefaultRetryPeriod
	renewDeadline = controller.DefaultRenewDeadline
	termLimit     = controller.DefaultTermLimit

	provisionerName = "gentics.com/zfs"
)

func main() {
	viper.SetEnvPrefix("zfs")
	viper.AutomaticEnv()

	viper.SetDefault("zpool_mount_prefix", "/")
	viper.SetDefault("zpool", "storage")
	viper.SetDefault("parent_dataset", "kubernetes/pv")
	viper.SetDefault("share_subnet", "10.0.0.0/8")
	viper.SetDefault("share_options", "")
	viper.SetDefault("server_hostname", "")
	viper.SetDefault("kube_conf", "kube.conf")
	viper.SetDefault("kube_reclaim_policy", "Delete")
	viper.SetDefault("debug", false)

	if viper.GetBool("debug") == true {
		log.SetLevel(log.DebugLevel)
	}

	config, err := clientcmd.BuildConfigFromFlags("", viper.GetString("kube_conf"))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to build config")
	}
	log.WithFields(log.Fields{
		"config": viper.GetString("kube_conf"),
	}).Info("Loaded kubernetes config")

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to create client")
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Failed to get server version")
	}
	log.WithFields(log.Fields{
		"version": serverVersion.GitVersion,
	}).Info("Retrieved server version")

	if viper.GetString("server_hostname") == "" {
		hostname, err := exec.Command("hostname", "-f").Output()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Determining server hostname via \"hostname -f\" failed")
		}
		viper.Set("server_hostname", hostname)
	}

	// Create the provisioner and start the controller
	zfsProvisioner := provisioner.NewZFSProvisioner(viper.GetString("zpool"), viper.GetString("zpool_mount_prefix"), viper.GetString("parent_dataset"), viper.GetString("share_options"), viper.GetString("share_subnet"), viper.GetString("server_hostname"), viper.GetString("kube_reclaim_policy"))
	pc := controller.NewProvisionController(clientset, 15*time.Second, provisionerName, zfsProvisioner, serverVersion.GitVersion, false, 2, leasePeriod, renewDeadline, retryPeriod, termLimit)
	log.Info("Listening for events")
	pc.Run(wait.NeverStop)
}

func validateProvisionerName(provisioner string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if len(provisioner) == 0 {
		allErrs = append(allErrs, field.Required(fldPath, provisioner))
	}
	if len(provisioner) > 0 {
		for _, msg := range validation.IsQualifiedName(strings.ToLower(provisioner)) {
			allErrs = append(allErrs, field.Invalid(fldPath, provisioner, msg))
		}
	}
	return allErrs
}
