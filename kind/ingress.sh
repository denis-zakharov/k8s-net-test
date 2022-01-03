#!/bin/bash

# ingress
#wget https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml -O ingress-deploy.yaml
#for i in $(grep image: ingress-deploy.yaml | awk '{print $NF}' | sort -u); do docker pull $i; kind load docker-image $i; done
#rm -fv ingress-deploy.yaml

kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# wait ingress
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=120s
