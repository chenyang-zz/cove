package auth

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/boxify/api-go/internal/infrastructure/storage"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

var SupportAvatarExts = []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

const MaxAvatarSize = 5 * 1024 * 1024 // 5MB

type UpdateAvatarLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

func NewUpdateAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAvatarLogic {
	return &UpdateAvatarLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.auth.updateavatar"),
	}
}

func (l *UpdateAvatarLogic) UpdateAvatar(userID uuid.UUID, input *request.FileRequest) (*response.UserResponse, error) {
	user, err := l.svcCtx.UserRepo.FindByID(l.ctx, userID)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(input.File.Filename))
	support := false
	for _, supportExt := range SupportAvatarExts {
		if supportExt == ext {
			support = true
			break
		}
	}
	if !support {
		return nil, xerr.BadRequestf("不支持的图谱类型: %s", ext)
	}

	if input.File.Size > MaxAvatarSize {
		return nil, xerr.BadRequest("头像大小不能超过 5MB 限制")
	}

	fileKey := storage.BuildFileKey(userID, "avatar", uuid.New(), ext)
	f, err := input.File.Open()
	if err != nil {
		return nil, xerr.Wrap(err, "读取头像文件出错")
	}
	defer f.Close()

	fileContent, err := io.ReadAll(f)
	if err != nil {
		return nil, xerr.Wrap(err, "读取头像文件出错")
	}

	err = l.svcCtx.Storage.Put(l.ctx, fileKey, fileContent)
	if err != nil {
		return nil, err
	}

	user.Avatar = &fileKey
	user, err = l.svcCtx.UserRepo.Update(l.ctx, user)
	if err != nil {
		return nil, err
	}

	return mapper.UserToResponse(user), nil
}
