package model

type Author struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateAuthorRequest struct {
	Name string `json:"name" binding:"required"`
}
