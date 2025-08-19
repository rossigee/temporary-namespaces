package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"istio.io/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig    string
	dryRun        bool
	annotationKey string

	systemNamespaces = []string{
		"default",
		"kube-system",
		"flux-system",
	}

	// Prometheus metrics
	namespacesChecked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "temporary_namespaces_checked_total",
			Help: "Total number of namespaces checked for expiry",
		},
		[]string{"result"},
	)

	namespacesDeleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "temporary_namespaces_deleted_total", 
			Help: "Total number of namespaces deleted",
		},
		[]string{"dry_run"},
	)

	cleanupErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "temporary_namespaces_errors_total",
			Help: "Total number of errors encountered during cleanup",
		},
		[]string{"type"},
	)

	cleanupDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "temporary_namespaces_cleanup_duration_seconds",
			Help: "Duration of namespace cleanup operations",
			Buckets: prometheus.DefBuckets,
		},
	)

	currentEligible = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "temporary_namespaces_current_eligible",
			Help: "Current number of namespaces eligible for deletion",
		},
	)
)

func init() {
	// Register Prometheus metrics
	prometheus.MustRegister(namespacesChecked)
	prometheus.MustRegister(namespacesDeleted)
	prometheus.MustRegister(cleanupErrors)
	prometheus.MustRegister(cleanupDuration)
	prometheus.MustRegister(currentEligible)
}

func isSystemNamespace(name string) bool {
	for _, ns := range systemNamespaces {
		if name == ns {
			return true
		}
	}
	return false
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	log.Infof("Starting metrics server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Errorf("Failed to start metrics server: %v", err)
	}
}

func main() {
	options := log.DefaultOptions()
	options.JSONEncoding = true
	if err := log.Configure(options); err != nil {
		fmt.Printf("{\"message\": \"unable to start logging system\", \"error\": \"%s\"}\n", err)
		os.Exit(1)
	}

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file (optional)")
	flag.BoolVar(&dryRun, "dry-run", false, "Enable dry run mode")
	flag.Parse()

	annotationKey = os.Getenv("KUBE_ANNOTATION_KEY")

	config, err := loadKubeConfig()
	if err != nil {
		log.Errorf("Error loading kubeconfig",
			"error", err,
		)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Error creating Kubernetes client",
			"error", err,
		)
		os.Exit(1)
	}

	// Start metrics server in a goroutine
	go startMetricsServer()

	for {
		if err := cleanupNamespaces(clientset); err != nil {
			log.Warnf("Error cleaning up namespaces",
				"error", err,
			)
			cleanupErrors.WithLabelValues("cleanup_failed").Inc()
		}

		// Sleep for an hour before running again
		time.Sleep(time.Hour)
	}
}

func loadKubeConfig() (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func cleanupNamespaces(clientset kubernetes.Interface) error {
	start := time.Now()
	defer func() {
		cleanupDuration.Observe(time.Since(start).Seconds())
	}()
	namespacesRegexStr := os.Getenv("NAMESPACES_REGEX")
	var namespacesRegex *regexp.Regexp
	if namespacesRegexStr != "" {
		var err error
		namespacesRegex, err = regexp.Compile(namespacesRegexStr)
		if err != nil {
			cleanupErrors.WithLabelValues("regex_compile").Inc()
			return fmt.Errorf("invalid NAMESPACES_REGEX: %v", err)
		}
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		cleanupErrors.WithLabelValues("list_namespaces").Inc()
		return err
	}

	currentTimestamp := time.Now().Unix()
	eligibleCount := 0

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			namespacesChecked.WithLabelValues("system_namespace").Inc()
			log.Warnf("Skipping system namespace",
				"namespace", ns.Name,
			)
			continue
		}

		if namespacesRegex != nil && !namespacesRegex.MatchString(ns.Name) {
			namespacesChecked.WithLabelValues("regex_no_match").Inc()
			log.Warnf("Skipping namespace that does not match configured regular expression",
				"namespace", ns.Name,
			)
			continue
		}

		annotations := ns.Annotations
		expiryTimestampStr := annotations[annotationKey]

		if expiryTimestampStr == "" {
			namespacesChecked.WithLabelValues("no_annotation").Inc()
			continue
		}

		expiryTimestamp, err := strconv.ParseInt(expiryTimestampStr, 10, 64)
		if err != nil {
			namespacesChecked.WithLabelValues("parse_error").Inc()
			cleanupErrors.WithLabelValues("parse_timestamp").Inc()
			log.Warnf("Error parsing expiry timestamp in namespace",
				"err", err,
				"namespace", ns.Name,
			)
			continue
		}

		if expiryTimestamp > currentTimestamp {
			namespacesChecked.WithLabelValues("not_expired").Inc()
			continue
		}

		eligibleCount++
		namespacesChecked.WithLabelValues("expired").Inc()

		if dryRun {
			namespacesDeleted.WithLabelValues("true").Inc()
			log.Infof("(DRY-RUN) Namespace marked for deletion",
				"namespace", ns.Name,
			)
			continue
		}

		err = clientset.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
		if err != nil {
			cleanupErrors.WithLabelValues("delete_namespace").Inc()
			log.Errorf("Error deleting namespace",
				"err", err,
				"namespace", ns.Name,
			)
			continue
		}

		namespacesDeleted.WithLabelValues("false").Inc()
		log.Infof("Namespace deleted successfully",
			"namespace", ns.Name,
		)
	}

	// Update the current eligible gauge
	currentEligible.Set(float64(eligibleCount))

	return nil
}
