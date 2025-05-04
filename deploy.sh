#!/bin/bash

kubeconfig="session-manager/conf/kubeconfig.yaml"
context_list_as_string=$(kubectl config get-contexts -o name --kubeconfig session-manager/conf/kubeconfig.yaml)
context_list=($context_list_as_string)

key_file="session-manager/conf/key.pem"
cert_file="session-manager/conf/cert.pem"
host="localhost"

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout "$key_file" -out "$cert_file" -subj "/CN=$host/O=$host" \
 -addext "subjectAltName = DNS:$host"

for ctx in ${context_list[@]}; do
    echo "Configuring context: $ctx" 

    kubectl create secret tls tls --key "$key_file" --cert "$cert_file" --kubeconfig "$kubeconfig" --context "$ctx"

    KUBECONFIG="$kubeconfig" helm upgrade --install ingress-nginx ingress-nginx \
            --repo https://kubernetes.github.io/ingress-nginx \
            --namespace ingress-nginx \
            --create-namespace \
            --set controller.service.externalTrafficPolicy=Local \
            --kube-context "$ctx"

    kubectl wait --namespace ingress-nginx --for=condition=Ready pod --selector=app.kubernetes.io/component=controller \
      --timeout=120s --kubeconfig "$kubeconfig" --context "$ctx"

    if [ "$ctx" = "main" ]; then
        kubectl delete secret kubeconfig-secret --kubeconfig "$kubeconfig" --context "$ctx"
        kubectl delete configmap templates-config --kubeconfig "$kubeconfig" --context "$ctx"
        kubectl create secret generic kubeconfig-secret \
         --from-file=kubeconfig.yaml=./session-manager/conf/kubeconfig.yaml --kubeconfig "$kubeconfig" --context "$ctx"       
        kubectl create configmap templates-config --from-file=templates.yaml=./session-manager/conf/templates.yaml \
         --kubeconfig "$kubeconfig" --context "$ctx"

        kubectl apply -f session-manager/cluster-resources/ --kubeconfig "$kubeconfig" --context "$ctx"
    else
        kubectl apply -f session-manager/cluster-resources/ingress.yaml --kubeconfig "$kubeconfig" --context "$ctx" 
        kubectl apply -f session-manager/cluster-resources/ping-server.yaml --kubeconfig "$kubeconfig" --context "$ctx"
    fi

    (
      cd controllers/session
      make install KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
      make undeploy KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
      make deploy KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx"" \
       IMG="simaosantos1230212/session-controller"
    )
    (
      cd controllers/gc 
      make install KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
      make undeploy KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
      make deploy KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx"" \
       IMG="simaosantos1230212/gc-controller"
    )
    (
      cd controllers/network 
      make undeploy KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
      make deploy KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx"" \
       IMG="simaosantos1230212/network-controller"
    )  
done
