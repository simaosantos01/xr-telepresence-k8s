package handlers

import (
	"errors"
	"os"

	k8sClient "mr.telepresence/session-manager/k8s-client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"log"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

type Handler struct {
	clusterClientsetMap map[string]*kubernetes.Clientset
	clusterClientMap    map[string]*k8sClient.SessionClient
	sessionTemplates    map[string]*SessionTemplate
}

// Reference:
// 	- https://stackoverflow.com/questions/74537153/how-to-switch-kubernetes-contexts-dynamically-with-client-go
// 	- https://stackoverflow.com/questions/70885022/how-to-get-current-k8s-context-name-using-client-go

const kubeConfigPath = "conf/kubeconfig.yaml"
const templatesPath = "conf/templates.yaml"

func ConfigHandler() (*Handler, error) {
	var apiConfig *api.Config
	if apiConfig = clientcmd.GetConfigFromFileOrDie(kubeConfigPath); apiConfig == nil {
		return nil, errors.New("could not read config from file")
	}

	k8sClient.AddToScheme(scheme.Scheme)

	clusterClientMap := make(map[string]*k8sClient.SessionClient, len(apiConfig.Contexts))
	clusterClientsetMap := make(map[string]*kubernetes.Clientset, len(apiConfig.Contexts))

	// config main cluster clients if in cluster mode
	cfg, err := rest.InClusterConfig()
	if err == nil {
		client, err := k8sClient.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}
		clusterClientMap["main"] = client
		clusterClientsetMap["main"] = clientset
	}

	if err != nil {
		log.Println(err.Error())
	}

	// config remaning cluster clients

	if _, ok := clusterClientMap["main"]; ok {
		delete(apiConfig.Contexts, "main")
		log.Println("deleted")
	}

	for contextName := range apiConfig.Contexts {

		log.Println("context: ", contextName)
		cfg, err := buildConfigWithContext(contextName, kubeConfigPath)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		client, err := k8sClient.NewForConfig(cfg)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}

		clusterClientMap[contextName] = client
		clusterClientsetMap[contextName] = clientset
	}

	templates, err := readTemplates()
	if err != nil {
		return nil, err
	}

	return &Handler{
		clusterClientsetMap: clusterClientsetMap,
		clusterClientMap:    clusterClientMap,
		sessionTemplates:    templates}, nil
}

func buildConfigWithContext(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

// ref: https://medium.com/better-programming/parsing-and-creating-yaml-in-go-crash-course-2ec10b7db850

type SessionTemplateList struct {
	Templates []SessionTemplate `json:"templates"`
}

type SessionTemplate struct {
	Name                    string                                `json:"name"`
	SessionPodTemplates     corev1.PodTemplateList                `json:"sessionPodTemplates"`
	ClientPodTemplates      sessionv1alpha1.ClientPodTemplateList `json:"clientPodTemplates"`
	TimeoutSeconds          int                                   `json:"timeoutSeconds"`
	ReutilizeTimeoutSeconds int                                   `json:"reutilizeTimeoutSeconds"`
}

func readTemplates() (map[string]*SessionTemplate, error) {
	templatesMap := make(map[string]*SessionTemplate)

	f, err := os.ReadFile(templatesPath)
	if err != nil {
		return nil, err
	}

	var sessionTemplates SessionTemplateList
	if err := yaml.Unmarshal(f, &sessionTemplates); err != nil {
		return nil, err
	}

	for _, template := range sessionTemplates.Templates {
		templatesMap[template.Name] = &template
	}

	return templatesMap, err
}
