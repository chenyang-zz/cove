/**
 * @Time   : 2026/6/27 00:49
 * @Author : chenyangzhao542@gmail.com
 * @File   : health.go
 **/

package response

type HealthItem struct {
	ServerName string `json:"server_name"`
	IsHealthy  bool   `json:"is_healthy"`
	Error      string `json:"error,omitempty"`
}

type HealthResponse struct {
	List []*HealthItem `json:"list"`
}
