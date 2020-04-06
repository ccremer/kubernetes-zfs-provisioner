package main

import (
	"fmt"
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"

	"go.uber.org/zap"

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

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	err := viper.ReadInConfig()
	if err != nil {
		logger.Warn("Could not read config file", zap.Error(err))
	}
	config, err := clientcmd.BuildConfigFromFlags("", viper.GetString("kube_config_path"))
	if err != nil {
		logger.Fatal("Couldn't get in-cluster or kubectl config", zap.Error(err))
	}

	// Retrieve config und server serverVersion
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatal("Failed to create kubernetes client", zap.Error(err))
	}
	serverVersion, err := clientset.DiscoveryClient.ServerVersion()
	if err != nil {
		logger.Fatal("Failed retrieving server version", zap.Error(err))
	}
	logger.Info("Connected to cluster", zap.String("cluster", config.Host), zap.String("version", fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor)))
	p, err := provisioner.NewZFSProvisioner(logger)
	if err != nil {
		logger.Fatal("Failed to create ZFS provisioner", zap.Error(err))
	}

	// Start and export the prometheus collector
	handler := promhttp.HandlerFor(nil, promhttp.HandlerOpts{
		ErrorHandling: promhttp.PanicOnError,
	})
	http.Handle("/metrics", handler)
	logger.Info("Starting exporter")
	go func() {
		logger.Fatal("Failed to export metrics", zap.Error(http.ListenAndServe(":"+viper.GetString("metrics_port"), nil)))
	}()

	pc := controller.NewProvisionController(
		clientset,
		provisioner.Name,
		p,
		serverVersion.GitVersion,
	)
	logger.Info("Starting provisoner", zap.String("version", version), zap.String("commit", commit))
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
	viper.SetDefault("metrics_port", "8080")
	viper.SetDefault("kube_config_path", "")
}
