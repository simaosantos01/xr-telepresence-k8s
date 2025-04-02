# Session Manager

To run the session manager locally simply execute `$ go run . -k8sInClusterCfg=false`. This assumes that a `.kube/config` file exists on the OS home dir.

**To run this inside a Minikube cluster:**

Build docker image

```
$ docker build -t session_manager --no-cache -f ./Dockerfile ../
```

Load image into minikube cluster
```
$ minikube image load session_manager:latest
```

Set cluster permissions
```
$ cd clusterPermissions
$ kubectl apply -f session-manager-sa.yaml
$ kubectl apply -f session-manager-role.yaml
$ kubectl apply -f session-manager-rolebinding.yaml
```

Create the pod in minikube cluster
```
$ kubectl run --rm -i session-manager --image=session_manager:latest --overrides='{ "spec": { "serviceAccount": "session-manager-sa" }  }' --image-pull-policy=Never
```

Exposing the pod
```
$ kubectl expose pod session-manager --type=NodePort --port=8080 --name=sessionmanager-service
$ minikube service sessionmanager-service
```



