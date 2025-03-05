package handlers

type SessionId struct {
	Name string `json:"name" binding:"required"`
}
