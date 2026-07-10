/*
 * @Time   : 2026-07-10 12:04:51
 * @Author : chenyang
 * @File   : toolconfig.go
 */
package request

type UriToolKeyRequest struct {
	ToolKey string `uri:"tool_key" binding:"required"`
}

type ToggleToolRequest struct {
	UriToolKeyRequest
	Enabled bool `json:"enabled" binding:"required"`
}
