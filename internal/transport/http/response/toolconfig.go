/*
 * @Time   : 2026-07-10 12:05:36
 * @Author : chenyang
 * @File   : toolconfig.go
 */

package response

type ToolConfigResponse struct {
	ToolKey     string `json:"tool_key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	ToolType    string `json:"tool_type"`
	NeedsConfig bool   `json:"needs_config"`
	ConfigHit   string `json:"config_hit"`
	Enabled     bool   `json:"enabled"`
}
