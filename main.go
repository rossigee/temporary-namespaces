package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

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
)

func isSystemNamespace(name string) bool {
	for _, ns := range systemNamespaces {
		if name == ns {
			return true
		}
	}
	return false
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

	for {
		if err := cleanupNamespaces(clientset); err != nil {
			log.Warnf("Error cleaning up namespaces",
				"error", err,
			)
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
	namespacesRegexStr := os.Getenv("NAMESPACES_REGEX")
	var namespacesRegex *regexp.Regexp
	if namespacesRegexStr != "" {
		var err error
		namespacesRegex, err = regexp.Compile(namespacesRegexStr)
		if err != nil {
			return fmt.Errorf("invalid NAMESPACES_REGEX: %v", err)
		}
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	currentTimestamp := time.Now().Unix()

	for _, ns := range namespaces.Items {
		if isSystemNamespace(ns.Name) {
			log.Warnf("Skipping system namespace",
				"namespace", ns.Name,
			)
			continue
		}

		if namespacesRegex != nil && !namespacesRegex.MatchString(ns.Name) {
			log.Warnf("Skipping namespace that does not match configured regular expression",
				"namespace", ns.Name,
			)
			continue
		}

		annotations := ns.Annotations
		expiryTimestampStr := annotations[annotationKey]

		if expiryTimestampStr == "" {
			continue
		}

		expiryTimestamp, err := strconv.ParseInt(expiryTimestampStr, 10, 64)
		if err != nil {
			log.Warnf("Error parsing expiry timestamp in namespace",
				"err", err,
				"namespace", ns.Name,
			)
			continue
		}

		if expiryTimestamp > currentTimestamp {
			continue
		}
		if dryRun {
			log.Infof("(DRY-RUN) Namespace marked for deletion",
				"namespace", ns.Name,
			)
			continue
		}

		err = clientset.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Errorf("Error deleting namespace",
				"err", err,
				"namespace", ns.Name,
			)
			continue
		}

		log.Infof("Namespace deleted successfully",
			"namespace", ns.Name,
		)
	}

	return nil
}
