package main

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCleanupNamespaces(t *testing.T) {
	// Create a fake clientset for testing
	clientset := fake.NewSimpleClientset()

	// Set annotationKey to match main function behavior
	annotationKey = "your-annotation-key"

	// Define a test namespace with an expiry timestamp
	testNamespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Annotations: map[string]string{
				"your-annotation-key": strconv.FormatInt(time.Now().Unix()-3600, 10), // Set an expired timestamp
			},
		},
	}

	// Create the test namespace in the fake clientset
	_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error creating test namespace: %v", err)
	}

	// Call the cleanupNamespaces function
	err = cleanupNamespaces(clientset)
	if err != nil {
		t.Fatalf("Error cleaning up namespaces: %v", err)
	}

	// Check if the test namespace was deleted
	_, err = clientset.CoreV1().Namespaces().Get(context.TODO(), "test-namespace", metav1.GetOptions{})
	if !errors.IsNotFound(err) {
		t.Errorf("Test namespace was not deleted as expected")
	}
}

func TestAvoidDeletingSystemAndNonMatchingNamespaces(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	annotationKey = "your-annotation-key"

	systemNamespaces := []string{"default", "kube-system", "flux-system"}
	otherNamespaces := []string{"other-namespace", "test-namespace"}
	allNamespaces := append(systemNamespaces, otherNamespaces...)

	for _, ns := range allNamespaces {
		testNamespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Annotations: map[string]string{
					annotationKey: strconv.FormatInt(time.Now().Unix()-3600, 10),
				},
			},
		}
		_, err := clientset.CoreV1().Namespaces().Create(context.TODO(), testNamespace, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("Error creating namespace %s: %v", ns, err)
		}
	}

	// Set regex to match only "test-namespace"
	os.Setenv("NAMESPACES_REGEX", "test-.*")

	if err := cleanupNamespaces(clientset); err != nil {
		t.Fatalf("Error cleaning up namespaces: %v", err)
	}

	for _, ns := range allNamespaces {
		_, err := clientset.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{})
		if ns == "test-namespace" {
			if !errors.IsNotFound(err) {
				t.Errorf("Namespace '%s' was not deleted as expected", ns)
			}
		} else {
			if err != nil {
				t.Errorf("Namespace '%s' was deleted but shouldn't have been", ns)
			}
		}
	}
}
