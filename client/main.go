package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file; KUBECONFIG overrides this")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file; KUBECONFIG overrides this")
	}
	manifest := flag.String("manifest", "manifest.yaml", "deploy pinger manifest: deployment, service, ingress")
	ns := flag.String("namespace", "default", "kubernetes namespace")
	flag.Parse()

	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		kubeconfig = &kubeconfigEnv
	}

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	must(err, "kubeconfig")
	clientset, err := kubernetes.NewForConfig(config)
	must(err, "clientset")

	y, err := os.Open(*manifest)
	must(err, "open manifest")
	defer y.Close()

	var deployment appsv1.Deployment
	var service corev1.Service
	var ingress networkingv1.Ingress

	decoder := yaml.NewYAMLOrJSONDecoder(y, 4096)
	err = decoder.Decode(&deployment)
	must(err, "yaml decode deployment")
	err = decoder.Decode(&service)
	must(err, "yaml decode service")
	err = decoder.Decode(&ingress)
	if err != io.EOF {
		must(err, "yaml decode ingress")
	}

	ctx := context.Background()
	namespace := *ns

	_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, &deployment, metav1.CreateOptions{})
	must(err, "create deployment")

	_, err = clientset.CoreV1().Services(namespace).Create(ctx, &service, metav1.CreateOptions{})
	must(err, "create service")

	_, err = clientset.NetworkingV1().Ingresses(namespace).Create(ctx, &ingress, metav1.CreateOptions{})
	must(err, "create ingress")
}

func must(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
