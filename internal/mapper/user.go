/**
 * @Time   : 2026/6/27 10:53
 * @Author : chenyangzhao542@gmail.com
 * @File   : user.go.go
 **/

package mapper

import (
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/transport/http/response"
)

func UserToResponse(user *models.User) *response.UserResponse {
	return &response.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Nickname:  user.Nickname,
		Email:     user.Email,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
