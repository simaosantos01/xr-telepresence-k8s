package handlers

type RegisterSessionBody struct {
	Name string `json:"name" binding:"required"`
}

type PatchClientsBody struct {
	SessionName string `json:"sessionName" binding:"required"`
	ClientName  string `json:"clientName" binding:"required"`
}
