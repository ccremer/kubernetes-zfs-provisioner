package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.uber.org/zap"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

const (
	leasePeriod   = controller.DefaultLeaseDuration
	retryPeriod   = controller.DefaultRetryPeriod
	renewDeadline = controller.DefaultRenewDeadline
)

func main() {
	viper.SetEnvPrefix("zfs")
	viper.AutomaticEnv()
	viper.SetDefault("metrics_port", "8080")

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Try in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// Try current kubectl context
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			logger.Fatal("Couldn't get in-cluster or kubectl config", zap.Error(err))
		}
		logger.Info("Succeeded with kubectl config")
	} else {
		logger.Info("Succeeded with in-cluster config")
	}

	// Retrieve config und server version
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatal("Failed to create kubernetes client", zap.Error(err))
	}
	version, err := clientset.DiscoveryClient.ServerVersion()
	if err != nil {
		logger.Fatal("Failed retrieving server version", zap.Error(err))
	}
	logger.Info("Connected to cluster", zap.String("cluster", config.Host), zap.String("version", fmt.Sprintf("%s.%s", version.Major, version.Minor)))
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
		version.GitVersion,
	)
	logger.Info("Starting provisoner")
	pc.Run(wait.NeverStop)
}
