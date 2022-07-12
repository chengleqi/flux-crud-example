package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

func main() {
	// register the GitOps Toolkit schema definitions
	scheme := runtime.NewScheme()
	_ = sourcev1.AddToScheme(scheme)
	_ = helmv2.AddToScheme(scheme)

	// init Kubernetes client
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}

	// set a deadline for the Kubernetes API operations
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// create a Helm repository pointing to Bitnami
	helmRepository := &sourcev1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bitnami",
			Namespace: "default",
		},
		Spec: sourcev1.HelmRepositorySpec{
			URL: "https://charts.bitnami.com/bitnami",
			Interval: metav1.Duration{
				Duration: 30 * time.Minute,
			},
		},
	}
	if err := kubeClient.Create(ctx, helmRepository); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("HelmRepository bitnami created")
	}
}
