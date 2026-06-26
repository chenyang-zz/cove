package request

type CreateModelConfigRequest struct {
	Type      string `json:"type" binding:"required"`
	Provider  string `json:"provider" binding:"required"`
	Model     string `json:"model" binding:"required"`
	BaseURL   string `json:"base_url" binding:"required"`
	APIKey    string `json:"api_key" binding:"required"`
	IsDefault bool   `json:"is_default"`
}
