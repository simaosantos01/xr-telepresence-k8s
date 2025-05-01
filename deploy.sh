#!/bin/bash

kubeconfig="session-manager/conf/kubeconfig.yaml"
context_list_as_string=$(kubectl config get-contexts -o name --kubeconfig session-manager/conf/kubeconfig.yaml)
context_list=($context_list_as_string)

for ctx in ${context_list[@]}; do
    echo "Configuring context: $ctx" 
    
    KUBECONFIG="$kubeconfig" helm upgrade --install ingress-nginx ingress-nginx \
            --repo https://kubernetes.github.io/ingress-nginx \
            --namespace ingress-nginx \
            --create-namespace \
            --kube-context "$ctx"

    if [ "$ctx" = "main" ]; then
        kubectl apply -f session-manager/cluster-resources/ --kubeconfig "$kubeconfig" --context "$ctx"
    else
        kubectl apply -f session-manager/cluster-resources/ingress.yaml --kubeconfig "$kubeconfig" --context "$ctx" 
        kubectl apply -f session-manager/cluster-resources/ping-server.yaml --kubeconfig "$kubeconfig" --context "$ctx"
    fi

    (
      cd controllers/session && make install KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
    )
    (
      cd controllers/gc && make install KUBECTL="kubectl --kubeconfig="../../$kubeconfig" --context="$ctx""
    ) 
done
