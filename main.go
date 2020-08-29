package main

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"k8s.io/klog"
	"net/http"
	"strings"

	"github.com/knadh/koanf"
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
	koanfInstance = koanf.New(".")
)

func main() {
	loadDefaultValues()
	loadEnvironmentVariables()

	config, err := clientcmd.BuildConfigFromFlags("", koanfInstance.String("kube_config_path"))
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

	instance := koanfInstance.String(provisionerInstanceKey)
	klog.Infof("Connected to cluster \"%s\" version \"%s.%s\"", config.Host, serverVersion.Major, serverVersion.Minor)
	p, err := provisioner.NewZFSProvisioner(instance)
	if err != nil {
		klog.Fatalf("Failed to create ZFS provisioner: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})

	pc := controller.NewProvisionController(
		clientset,
		instance,
		p,
		serverVersion.GitVersion,
		controller.MetricsAddress(koanfInstance.String(metricsAddrKey)),
		controller.MetricsPort(int32(koanfInstance.Int(metricsPortKey))),
	)

	klog.Infof("Starting provisioner version \"%s\" commit \"%s\"", version, commit)
	pc.Run(wait.NeverStop)
}

func loadDefaultValues() {
	_ = koanfInstance.Load(confmap.Provider(map[string]interface{}{
		metricsPortKey:         "8080",
		metricsAddrKey:         "0.0.0.0",
		kubeConfigPathKey:      "",
		provisionerInstanceKey: "pv.kubernetes.io/zfs",
	}, "."), nil)
}

func loadEnvironmentVariables() {
	prefix := "ZFS_"
	err := koanfInstance.Load(env.Provider(prefix, ".", func(s string) string {
		s = strings.TrimPrefix(s, prefix)
		s = strings.Replace(strings.ToLower(s), "_", "-", -1)
		return s
	}), nil)
	if err != nil {
		klog.Fatalf("Could not load environment variables: %w", err)
	}
}
