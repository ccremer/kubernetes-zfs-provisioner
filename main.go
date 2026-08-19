package main

import (
	"context"
	"fmt"
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v13/controller"
)

const (
	metricsAddrKey         = "METRICS_ADDR"
	metricsPortKey         = "METRICS_PORT"
	kubeConfigPathKey      = "KUBE_CONFIG_PATH"
	provisionerInstanceKey = "PROVISIONER_INSTANCE"
)

type Settings struct {
	MetricsAddr         string
	MetricsPort         int
	KubeConfigPath      string
	ProvisionerInstance string
}

var (
	// These will be populated by Goreleaser at build time
	version = "snapshot"
	commit  = "dirty"

	settings Settings
)

func main() {
	loadEnvironmentVariables()

	log := klog.NewKlogr()

	log.Info("Using configuration", "config", settings)

	config, err := clientcmd.BuildConfigFromFlags("", settings.KubeConfigPath)
	if err != nil {
		klog.Fatalf("Couldn't get in-cluster or kubectl config: %v", err)
	}

	// Retrieve config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Failed to create kubernetes client: %v", err)
	}

	log.Info("Connected to cluster", "host", config.Host)
	p, err := provisioner.NewZFSProvisioner(settings.ProvisionerInstance, log)
	if err != nil {
		klog.Fatalf("Failed to create ZFS provisioner: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
	})

	ctx := klog.NewContext(context.Background(), log)
	pc := controller.NewProvisionController(
		ctx,
		clientset,
		settings.ProvisionerInstance,
		p,
		controller.MetricsAddress(settings.MetricsAddr),
		controller.MetricsPort(int32(settings.MetricsPort)),
	)

	// Probe listener is independent of leader election. The library only
	// binds :8080 after it wins the lease; standby replicas must still be Ready.
	go func() {
		mux := http.NewServeMux()
		ok := func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}
		mux.HandleFunc("/healthz", ok)
		mux.HandleFunc("/readyz", ok)
		addr := fmt.Sprintf("%s:%d", settings.MetricsAddr, settings.MetricsPort+1)
		log.Info("starting healthz", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			klog.ErrorS(err, "healthz server exited")
		}
	}()

	// The expander writes to the backend (zfs set refquota/refreservation) and to
	// PV/PVC objects, so exactly one replica may run it - just like Provision and
	// Delete, which the library already guards with leader election. The library
	// does not expose its leadership, so the expander holds its own lease.
	go runExpander(ctx, clientset, p, log)

	log.Info("Starting provisioner", "version", version, "commit", commit)
	pc.Run(ctx)
}

// runExpander runs the volume expander under a dedicated lease so that only the
// leader among the replicas performs expansion. Outside a cluster (no namespace
// to hold the lease) it runs unguarded, which is fine for a single dev process.
func runExpander(ctx context.Context, clientset kubernetes.Interface, p *provisioner.ZFSProvisioner, log klog.Logger) {
	ns := expanderNamespace()
	if ns == "" {
		log.Info("no namespace for expander leader election; running the expander without a lease")
		provisioner.RunExpander(ctx, clientset, p, log)
		return
	}
	identity, _ := os.Hostname()
	if identity == "" {
		identity = "zfs-provisioner"
	}
	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: "zfs-provisioner-expander", Namespace: ns},
		Client:     clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: identity},
	}
	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) { provisioner.RunExpander(ctx, clientset, p, log) },
			OnStoppedLeading: func() { log.Info("expander lost leadership; standing by") },
		},
	})
}

// expanderNamespace is the namespace the expander lease lives in: the pod's own
// namespace, from the downward-API env var or the service-account mount.
func expanderNamespace() string {
	if ns := strings.TrimSpace(os.Getenv("POD_NAMESPACE")); ns != "" {
		return ns
	}
	if b, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return strings.TrimSpace(string(b))
	}
	return ""
}

func loadEnvironmentVariables() {
	prefix := "ZFS_"

	defaults := map[string]string{
		metricsPortKey:         "8080",
		metricsAddrKey:         "0.0.0.0",
		kubeConfigPathKey:      "",
		provisionerInstanceKey: "pv.kubernetes.io/zfs",
	}

	for key, _ := range defaults {
		value, found := os.LookupEnv(fmt.Sprintf("%s%s", prefix, key))
		if found {
			defaults[key] = value
		}
	}
	settings = Settings{
		MetricsAddr:         defaults[metricsAddrKey],
		MetricsPort:         parseInt(defaults[metricsPortKey]),
		KubeConfigPath:      defaults[kubeConfigPathKey],
		ProvisionerInstance: defaults[provisionerInstanceKey],
	}
}

func parseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		klog.Fatalf("Failed to convert metrics port to integer: %v", err)
	}
	return i
}
