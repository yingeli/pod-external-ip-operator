# Pod External IP Operator

The Pod External IP Operator is an implementation of a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) which could associate specified external IP to a pod.

## Getting started

Firstly, ensure you have an VM-based AKS cluster (VMSS is not supported) with CNI networking. The public IP address resources that will be used for egress traffic needs to be created in the node resource group of AKS. You need to create an Azure service principle which will be used by the egress-ip-operator and add it as Contrinutor of the AKS node resource group. And you need to make sure the 2 NSGs of the AKS cluster allow inbound traffic.

Install cert-manager:
```
$ kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
```

Deploy pod-external-ip-operator into your AKS cluster:
```
git clone https://github.com/yingeli/pod-external-ip-operator.git
cd pod-external-ip-operator
make deploy IMG=yingeli/pod-external-ip-operator:0.1.67
```

A namespace "pod-external-ip" will be created during the deployment. Now we add the credential of the Azure service principle as secrets:
```
kubectl create secret generic azure-credential --namespace=pod-external-ip --from-literal='clientid=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' --from-literal='clientsecret=xxxxxxxxxxxxxxxxxxxxxxxxxxxxx' --from-literal='tenantid=xxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxxx'
```

To associate with an public ip, add annotation to your pod:
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-001
  labels:
    app: nginx-001
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx-001
  template:
    metadata:
      annotations:
        podexternalip.yglab.eu.org/externalip: 65.52.164.56    
      labels:
        app: nginx-001
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
```
