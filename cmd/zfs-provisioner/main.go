package main

import (
	"errors"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"git.gentics.com/psc/kubernetes-zfs-provisioner/pkg/provisioner"
	log "github.com/Sirupsen/logrus"
	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/simt2/go-zfs"
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
)

func main() {
	viper.SetEnvPrefix("zfs")
	viper.AutomaticEnv()

	viper.SetDefault("parent_dataset", "")
	viper.SetDefault("share_subnet", "10.0.0.0/8")
	viper.SetDefault("share_options", "")
	viper.SetDefault("server_hostname", "")
	viper.SetDefault("kube_conf", "kube.conf")
	viper.SetDefault("kube_reclaim_policy", "Delete")
	viper.SetDefault("provisioner_name", "gentics.com/zfs")
	viper.SetDefault("metrics_port", "8080")
	viper.SetDefault("debug", false)

	if viper.GetBool("debug") == true {
		log.SetLevel(log.DebugLevel)
	}

	// Ensure provisioner name is valid
	if errs := validateProvisionerName(viper.GetString("provisioner_name"), field.NewPath("provisioner")); len(errs) != 0 {
		log.WithFields(log.Fields{
			"errors": errs,
		}).Fatal("Invalid provisioner name specified")
	}

	// Retrieve kubernetes config and connect to server
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

	// Determine hostname if not set
	if viper.GetString("server_hostname") == "" {
		hostname, err := exec.Command("hostname", "-f").Output()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Fatal("Determining server hostname via \"hostname -f\" failed")
		}
		viper.Set("server_hostname", hostname)
	}

	// Load ZFS parent dataset
	if viper.GetString("parent_dataset") == "" {
		log.WithFields(log.Fields{
			"error": errors.New("Parent dataset is not set"),
		}).Fatal("Could not open ZFS parent dataset")
	}
	parent, err := zfs.GetDataset(viper.GetString("parent_dataset"))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Could not open ZFS parent dataset")
	}

	// Create the provisioner
	zfsProvisioner := provisioner.NewZFSProvisioner(parent, viper.GetString("share_options"), viper.GetString("share_subnet"), viper.GetString("server_hostname"), viper.GetString("kube_reclaim_policy"))

	// Start and export the prometheus collector
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(zfsProvisioner)
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog:      log.StandardLogger(),
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
	http.Handle("/metrics", handler)
	go func() {
		log.WithFields(log.Fields{
			"error": http.ListenAndServe(":"+viper.GetString("metrics_port"), nil),
		}).Error("Prometheus exporter failed")
	}()
	log.Info("Started Prometheus exporter")

	// Start the controller
	pc := controller.NewProvisionController(clientset, 15*time.Second, viper.GetString("provisioner_name"), zfsProvisioner, serverVersion.GitVersion, false, 2, leasePeriod, renewDeadline, retryPeriod, termLimit)
	log.Info("Listening for events")
	pc.Run(wait.NeverStop)
}

// validateProvisioner tests if provisioner is a valid qualified name.
// https://github.com/kubernetes/kubernetes/blob/release-1.4/pkg/apis/storage/validation/validation.go
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
