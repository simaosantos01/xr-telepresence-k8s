package handlers

type RegisterSessionBody struct {
	TemplateName       string `json:"templateName" binding:"required"`
	SessionPodsCluster string `json:"sessionPodsCluster" binding:"required"`
}

type CreateClientBody struct {
	ClientId string `json:"clientId" binding:"required"`
	Cluster  string `json:"cluster" binding:"required"`
}

type UpdateClientBody struct {
	Connected *bool `json:"connected" binding:"required"`
}
