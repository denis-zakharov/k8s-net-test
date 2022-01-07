package main

import (
	"log"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type controller struct {
	selector  labels.Selector
	replicas  int32
	waitReady chan struct{}
}

func (c *controller) decodeDeployment(obj interface{}) (*appsv1.Deployment, bool) {
	var deployment *appsv1.Deployment
	var ok bool
	if deployment, ok = obj.(*appsv1.Deployment); !ok {
		return nil, false
	}

	ls := labels.Set(deployment.ObjectMeta.Labels)
	if !c.selector.Matches(ls) {
		return nil, false
	}
	return deployment, true
}

func (c *controller) addDeploymentHandler(obj interface{}) {
	if deployment, ok := c.decodeDeployment(obj); ok {
		log.Printf("Deployment %s has beed created", deployment.ObjectMeta.Name)
	}
}

func (c *controller) updateDeploymentHandler(oldObj, newObj interface{}) {
	deployment, ok := c.decodeDeployment(newObj)
	if !ok {
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

func (c *controller) deleteDeploymentHandler(obj interface{}) {
	if deployment, ok := c.decodeDeployment(obj); ok {
		log.Printf("Deployment %s has beed deleted", deployment.ObjectMeta.Name)
	}
}
