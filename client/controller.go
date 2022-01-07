package main

import (
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

type controller struct {
	selector  labels.Selector
	replicas  int32
	waitReady chan struct{}
}

func (c *controller) addDeploymentHandler(obj interface{}) {
	var deployment *appsv1.Deployment
	var ok bool
	if deployment, ok = obj.(*appsv1.Deployment); !ok {
		utilruntime.HandleError(fmt.Errorf("error decoding deployment, invalid type"))
		return
	}

	ls := labels.Set(deployment.GetObjectMeta().GetLabels())
	if !c.selector.Matches(ls) {
		return
	}
	if deployment.Status.AvailableReplicas == c.replicas {
		log.Printf("All pods %d/%d in the deployment are ready",
			deployment.Status.AvailableReplicas, c.replicas)
		close(c.waitReady)
	} else {
		log.Printf("Pods are starting up: %d/%d in the deployment are ready",
			deployment.Status.AvailableReplicas, c.replicas)
	}
}

func (c *controller) updateDeploymentHandler(oldObj, newObj interface{}) {
	c.addDeploymentHandler(newObj)
}

func (c *controller) deleteDeploymentHandler(obj interface{}) {
	log.Println("Deployment has been deleted")
}
