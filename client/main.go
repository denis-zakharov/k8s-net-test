package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/denis-zakharov/k8s-net-test/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	namespace := flag.String("namespace", "default", "kubernetes namespace")
	numReplicas := flag.Int("replicas", 2, "a number of replicas")
	flag.Parse()

	replicas := int32(*numReplicas)

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
	deployment.Spec.Replicas = &replicas
	err = decoder.Decode(&service)
	must(err, "yaml decode service")
	err = decoder.Decode(&ingress)
	if err != io.EOF {
		must(err, "yaml decode ingress")
	}

	// TODO context with timeout
	ctx := context.Background()
	// ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	// defer cancel()
	ns := *namespace

	_, err = clientset.AppsV1().Deployments(ns).Create(ctx, &deployment, metav1.CreateOptions{})
	must(err, "create deployment")

	_, err = clientset.CoreV1().Services(ns).Create(ctx, &service, metav1.CreateOptions{})
	must(err, "create service")

	_, err = clientset.NetworkingV1().Ingresses(ns).Create(ctx, &ingress, metav1.CreateOptions{})
	must(err, "create ingress")

	// wait for all pods ready
	podSelector := labels.SelectorFromSet(deployment.Spec.Selector.MatchLabels)
	deploymentSelector := labels.SelectorFromSet(deployment.ObjectMeta.Labels)
	waitController := make(chan struct{})
	controller := &controller{deploymentSelector, replicas, waitController}

	informerFactory := informers.NewSharedInformerFactory(clientset, 10*time.Minute)
	deploymentInformer := informerFactory.Apps().V1().Deployments()
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addDeploymentHandler,
		UpdateFunc: controller.updateDeploymentHandler,
		DeleteFunc: controller.deleteDeploymentHandler,
	})
	waitInformer := make(chan struct{})
	defer close(waitInformer)
	informerFactory.Start(waitInformer)
	informerFactory.WaitForCacheSync(waitInformer)

	<-waitController // pods are ready

	// create a payload for /direct handler
	pods, err := clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: podSelector.String()})
	if err != nil {
		must(err, "cannot find test pods")
	}
	directPayload := make([]model.DirectReqPayloadItem, len(pods.Items))
	for i, pod := range pods.Items {
		name := pod.ObjectMeta.Name
		directPayload[i].Hostname = name
		podIPs := pod.Status.PodIPs
		addrs := make([]string, len(podIPs))
		for i, ip := range podIPs {
			addrs[i] = ip.IP
		}
		directPayload[i].Addrs = addrs
	}

	// create a payload for /svc handler
	svcName := service.ObjectMeta.Name
	svcPort := service.Spec.Ports[len(service.Spec.Ports)-1].Port
	svcURL := fmt.Sprintf("http://%s:%d", svcName, svcPort)
	svcPayload := model.SvcReqPayload{SvcURL: svcURL, Count: int(100 * replicas)}

	// TODO collect ingress info
	ingressURL := "http://localhost:9080" // KIND ingress

	// run svc and pod-to-pod checks
	checker := NewChecker()
	errc := make(chan error)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		errc <- checker.Svc(ingressURL, &svcPayload)
		wg.Done()
	}()
	go func() {
		errc <- checker.Direct(ingressURL, directPayload)
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(errc)
	}()

	for err := range errc {
		logErr(err, "net check")
	}

	// tear down
	logErr(clientset.NetworkingV1().Ingresses(ns).Delete(ctx, ingress.ObjectMeta.Name, metav1.DeleteOptions{}), "ingress")
	logErr(clientset.CoreV1().Services(ns).Delete(ctx, service.ObjectMeta.Name, metav1.DeleteOptions{}), "service")
	logErr(clientset.AppsV1().Deployments(ns).Delete(ctx, deployment.ObjectMeta.Name, metav1.DeleteOptions{}), "deployment")
}

func must(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func logErr(err error, prefix string) {
	if err != nil {
		log.Printf("[ERROR] %s: %s\n", prefix, err.Error())
	}
}
