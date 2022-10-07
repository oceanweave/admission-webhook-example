#! /bin/sh

kubectl delete -f auto-deploy.yaml
kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io admission-registry
kubectl delete mutatingwebhookconfigurations.admissionregistration.k8s.io admission-registry-mutate
