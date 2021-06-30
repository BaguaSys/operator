# Kubernetes operator for Bagua jobs

This repository implements a kubernetes operator for Bagua distributed training job which supports static and elastic workloads. See [CRD definition](https://github.com/BaguaSys/operator/blob/preonline/config/crd/bases/bagua.kuaishou.com_baguas.yaml).

### Prerequisites
- Kubernetes
- kubectl


### Installation
- Deploy operator locally
```shell

git clone https://github.com/BaguaSys/operator.git
cd operator

kubectl apply -f config/crd/bases/bagua.kuaishou.com_baguas.yaml

go run ./main.go
```


### Examples
You can get demos in `config/samples`, and run as follows,
- static mode
```shell

kubectl apply -f config/samples/bagua_v1alpha1_bagua_static.yaml
```
Verify pods are running
```yaml

kubectl get pods

NAME                           READY   STATUS    RESTARTS   AGE
bagua-sample-static-master-0   1/1     Running   0          45s
bagua-sample-static-worker-0   1/1     Running   0          45s
bagua-sample-static-worker-1   1/1     Running   0          45s
```

- elastic mode
```shell

kubectl apply -f config/samples/bagua_v1alpha1_bagua_elastic.yaml
```
Verify pods are running
```yaml

kubectl get pods

NAME                            READY   STATUS    RESTARTS   AGE
bagua-sample-elastic-etcd-0     1/1     Running   0          63s
bagua-sample-elastic-worker-0   1/1     Running   0          63s
bagua-sample-elastic-worker-1   1/1     Running   0          63s
bagua-sample-elastic-worker-2   1/1     Running   0          63s
```