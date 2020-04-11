package main

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"k8s.io/klog"
	"net/http"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

const (
	metricsAddrKey         = "metrics_addr"
	metricsPortKey         = "metrics_port"
	kubeConfigPathKey      = "kube_config_path"
	provisionerInstanceKey = "provisioner_instance"
)

var (
	// These will be populated by Goreleaser at build time
	version = "snapshot"
	commit  = "dirty"
)

func main() {
	configureViper()

	config, err := clientcmd.BuildConfigFromFlags("", viper.GetString("kube_config_path"))
	if err != nil {
		klog.Fatalf("Couldn't get in-cluster or kubectl config: %v", err)
	}

	// Retrieve config und server version
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create kubernetes client: %v", err)
	}
	serverVersion, err := clientset.DiscoveryClient.ServerVersion()
	if err != nil {
		klog.Fatalf("Failed retrieving server version: %v", err)
	}
	klog.Infof("Connected to cluster \"%s\" version \"%s.%s\"", config.Host, serverVersion.Major, serverVersion.Minor)
	p, err := provisioner.NewZFSProvisioner()
	if err != nil {
		klog.Fatalf("Failed to create ZFS provisioner: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})

	pc := controller.NewProvisionController(
		clientset,
		viper.GetString(provisionerInstanceKey),
		p,
		serverVersion.GitVersion,
		controller.MetricsAddress(viper.GetString(metricsAddrKey)),
		controller.MetricsPort(viper.GetInt32(metricsPortKey)),
	)

	klog.Infof("Starting provisioner version \"%s\" commit \"%s\"", version, commit)
	pc.Run(wait.NeverStop)
}

func configureViper() {
	viper.SetEnvPrefix("zfs")
	viper.AutomaticEnv()
	viper.SetConfigName("zfs-provisioner")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/kubernetes")
	viper.AddConfigPath("/var/lib/kubernetes-zfs-provisioner")
	viper.AddConfigPath(".")
	viper.SetDefault(metricsPortKey, "8080")
	viper.SetDefault(metricsAddrKey, "0.0.0.0")
	viper.SetDefault(kubeConfigPathKey, "")
	viper.SetDefault(provisionerInstanceKey, "pv.kubernetes.io/zfs")

	err := viper.ReadInConfig()
	if err != nil {
		klog.Warningf("Could not read config file: %v", err)
	}
}
