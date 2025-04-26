package handlers

type RegisterSessionBody struct {
	TemplateName       string `json:"templateName" binding:"required"`
	SessionPodsCluster string `json:"sessionPodsCluster" binding:"required"`
}

type PatchClientsBody struct {
	SessionName string `json:"sessionName" binding:"required"`
	ClientName  string `json:"clientName" binding:"required"`
}
