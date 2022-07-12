package main

import (
	"context"
	"fmt"
	apimeta "github.com/fluxcd/pkg/apis/meta"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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

	// create a Helm release for nginx
	helmRelease := &helmv2.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
		},
		Spec: helmv2.HelmReleaseSpec{
			ReleaseName: "nginx",
			Interval: metav1.Duration{
				Duration: 5 * time.Minute,
			},
			Chart: helmv2.HelmChartTemplate{
				Spec: helmv2.HelmChartTemplateSpec{
					Chart:   "nginx",
					Version: "8.x",
					SourceRef: helmv2.CrossNamespaceObjectReference{
						Kind: sourcev1.HelmRepositoryKind,
						Name: "bitnami",
					},
				},
			},
			Values: &apiextensionsv1.JSON{Raw: []byte(`{"service": {"type": "ClusterIP"}}`)},
		},
	}

	if err := kubeClient.Create(ctx, helmRelease); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("HelmRelease nginx created")
	}

	// wait for the Helm release to be reconciled
	fmt.Println("Waiting for nginx to be installed")
	if err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (done bool, err error) {
		namespacedName := types.NamespacedName{
			Namespace: helmRelease.GetNamespace(),
			Name:      helmRelease.GetName(),
		}
		if err := kubeClient.Get(ctx, namespacedName, helmRelease); err != nil {
			return false, err
		}
		return meta.IsStatusConditionTrue(helmRelease.Status.Conditions, apimeta.ReadyCondition), nil
	}); err != nil {
		fmt.Println(err)
	}
	// print the reconciliation status
	fmt.Println(meta.FindStatusCondition(helmRelease.Status.Conditions, apimeta.ReadyCondition).Message)
}
