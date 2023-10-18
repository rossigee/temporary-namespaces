package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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
)

func main() {
	options := log.DefaultOptions()
	if err := log.Configure(options); err != nil {
		fmt.Printf("unable to start logging system: %s", err)
		os.Exit(1)
	}

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file (optional)")
	flag.BoolVar(&dryRun, "dry-run", false, "Enable dry run mode")
	flag.Parse()

	annotationKey = os.Getenv("KUBE_ANNOTATION_KEY")

	config, err := loadKubeConfig()
	if err != nil {
		log.Errorf("Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	for {
		if err := cleanupNamespaces(clientset); err != nil {
			log.Warnf("Error cleaning up namespaces: %v\n", err)
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

func cleanupNamespaces(clientset *kubernetes.Clientset) error {
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	currentTimestamp := time.Now().Unix()

	for _, ns := range namespaces.Items {
		annotations := ns.Annotations
		expiryTimestampStr := annotations[annotationKey]

		if expiryTimestampStr != "" {
			expiryTimestamp, err := strconv.ParseInt(expiryTimestampStr, 10, 64)
			if err != nil {
				log.Warnf("Error parsing expiry timestamp in namespace %s: %v\n", ns.Name, err)
				continue
			}

			if expiryTimestamp < currentTimestamp {
				if dryRun {
					log.Infof("Namespace %s marked for deletion (DRY RUN)\n", ns.Name)
				} else {
					err := clientset.CoreV1().Namespaces().Delete(context.TODO(), ns.Name, metav1.DeleteOptions{})
					if err != nil {
						log.Errorf("Error deleting namespace %s: %v\n", ns.Name, err)
					} else {
						log.Infof("Namespace %s deleted successfully\n", ns.Name)
					}
				}
			}
		}
	}

	return nil
}
