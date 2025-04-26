package handlers

import (
	"errors"
	k8sClient "mr.telepresence/session-manager/k8s-client"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

type Handler struct {
	clusterSessionClientMap map[string]*k8sClient.SessionClient
	sessionTemplates        map[string]*SessionTemplate
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

	clusterSessionClientMap := make(map[string]*k8sClient.SessionClient, len(apiConfig.Contexts))
	for contextName := range apiConfig.Contexts {
		kubeConfig, err := buildConfigWithContext(contextName, kubeConfigPath)
		if err != nil {
			return nil, err
		}

		client, err := k8sClient.NewForConfig(kubeConfig)
		if err != nil {
			return nil, err
		}

		clusterSessionClientMap[contextName] = client
	}

	templates, err := readTemplates()
	if err != nil {
		return nil, err
	}

	return &Handler{clusterSessionClientMap: clusterSessionClientMap, sessionTemplates: templates}, nil
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
