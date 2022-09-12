package main

import (
	"context"
	"fmt"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type CSVStatus struct {
	Phase string
}

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", "/etc/kubeconfig/kubeconfig")
	if err != nil {
		panic(err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	operatorChannel := os.Getenv("OPERATOR_CHANNEL")
	operatorApproval := os.Getenv("OPERATOR_APPROVAL")
	operatorName := os.Getenv("OPERATOR_NAME")
	operatorSource := os.Getenv("OPERATOR_SOURCE")
	operatorSourceNs := os.Getenv("OPERATOR_SOURCE_NS")
	operatorVersion := os.Getenv("OPERATOR_VERSION")

	resource := &unstructured.Unstructured{}
	resource.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "operators.coreos.com/v1alpha1",
		"kind":       "Subscription",
		"metadata": map[string]interface{}{
			"name":      operatorName,
			"namespace": "openshift-operators",
		},
		"spec": map[string]interface{}{
			"channel":             operatorChannel,
			"installPlanApproval": operatorApproval,
			"name":                operatorName,
			"source":              operatorSource,
			"sourceNamespace":     operatorSourceNs,
			"startingCSV":         operatorVersion,
		},
	})

	gvr := schema.GroupVersionResource{
		Group:    "operators.coreos.com",
		Version:  "v1alpha1",
		Resource: "subscriptions",
	}
	retry := 0
	for {
		_, err = dynamicClient.Resource(gvr).Namespace("openshift-operators").Create(context.TODO(), resource, metav1.CreateOptions{})

		if err != nil {
			retry++
			if retry == 10 {
				panic(err)
			}
			fmt.Printf("Failed to create subscription %q \n", err)
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	fmt.Printf("Created operator %q.\n", operatorName)

	gvr = schema.GroupVersionResource{
		Group:    "operators.coreos.com",
		Version:  "v1alpha1",
		Resource: "clusterserviceversions",
	}

	for {
		csv, err := dynamicClient.Resource(gvr).Namespace("openshift-operators").Get(context.TODO(), operatorVersion, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("Err getting csv %q.\n", err)
		} else {
			status := csv.Object["status"].(map[string]interface{})
			fmt.Printf("Current status phase %q.\n", status["phase"])
			if status["phase"] == "Succeeded" {
				break
			}
		}
		time.Sleep(10 * time.Second)
	}

}
