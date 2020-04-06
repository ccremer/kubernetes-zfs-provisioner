package main

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
	"net/http"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
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
	klog.V(2).Infof("Connected to cluster \"%s\" version \"%s.%s\"", config.Host, serverVersion.Major, serverVersion.Minor)
	p, err := provisioner.NewZFSProvisioner()
	if err != nil {
		klog.Fatalf("Failed to create ZFS provisioner: %v", err)
	}

	pc := controller.NewProvisionController(
		clientset,
		provisioner.Name,
		p,
		serverVersion.GitVersion,
	)

	go startMetricsExporter()
	klog.V(2).Infof("Starting provisoner version \"%s\" commit \"%s\"", version, commit)
	pc.Run(wait.NeverStop)
}

func startMetricsExporter() {
	// Start and export the prometheus collector
	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		ErrorHandling: promhttp.PanicOnError,
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})
	http.Handle("/metrics", handler)
	klog.V(3).Info("Starting exporter")
	bindAddr := ":" + viper.GetString("metrics_port")
	err := http.ListenAndServe(bindAddr, nil)
	if err != http.ErrServerClosed {
		klog.Errorf("Failed to export metrics: %v", err)
	}
}

func configureViper() {
	viper.SetEnvPrefix("zfs")
	viper.AutomaticEnv()
	viper.SetConfigName("zfs-provisioner")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/kubernetes")
	viper.AddConfigPath("/var/lib/kubernetes-zfs-provisioner")
	viper.AddConfigPath(".")
	viper.SetDefault("metrics_port", "8080")
	viper.SetDefault("kube_config_path", "")

	err := viper.ReadInConfig()
	if err != nil {
		klog.Warningf("Could not read config file: %v", err)
	}
}
