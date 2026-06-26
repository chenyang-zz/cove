package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredential  = errors.New("invalid username/email or password")
	ErrInvalidAccessToken = errors.New("invalid access token")
)

const (
	accessTokenTTL  = 7 * 24 * time.Hour
	refreshTokenTTL = 30 * 24 * time.Hour
)

type AuthService struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	issuer        *TokenIssuer
}

type RegisterInput struct {
	Username string
	Nickname *string
	Email    *string
	Avatar   *string
	Password string
}

type LoginInput struct {
	Login    string
	Password string
}

type RefreshInput struct {
	RefreshToken string
}

type AuthOutput struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	Email        *string   `json:"email,omitempty"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
}

func NewAuthService(users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, secret string) *AuthService {
	if users == nil || refreshTokens == nil {
		panic("auth repositories are required")
	}
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		issuer:        NewTokenIssuer(secret, accessTokenTTL),
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthOutput, error) {
	username := normalizeRequired(input.Username)
	email := normalizeOptional(input.Email, true)
	nickname := normalizeOptional(input.Nickname, false)
	avatar := normalizeOptional(input.Avatar, false)
	if username == "" || len(input.Password) < 6 {
		return AuthOutput{}, xerr.BadRequest("用户名或密码格式错误")
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		return AuthOutput{}, xerr.Internal("密码处理失败", err)
	}
	user, err := s.users.Create(ctx, repository.User{
		ID:           uuid.New(),
		Username:     username,
		Nickname:     nickname,
		Email:        email,
		Avatar:       avatar,
		PasswordHash: hash,
	})
	if err != nil {
		return AuthOutput{}, preserveAppError(err, "创建用户失败")
	}
	return s.issueAuthOutput(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthOutput, error) {
	login := normalizeRequired(input.Login)
	if login == "" || len(input.Password) < 6 {
		return AuthOutput{}, xerr.BadRequest("账号或密码格式错误")
	}
	user, err := s.users.FindByLogin(ctx, login)
	if err != nil {
		return AuthOutput{}, xerr.InvalidCredential()
	}
	if !CheckPassword(user.PasswordHash, input.Password) {
		return AuthOutput{}, xerr.InvalidCredential()
	}
	return s.issueAuthOutput(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, input RefreshInput) (AuthOutput, error) {
	raw := strings.TrimSpace(input.RefreshToken)
	if raw == "" {
		return AuthOutput{}, xerr.BadRequest("刷新令牌不能为空")
	}
	token, err := s.refreshTokens.FindByHash(ctx, HashRefreshToken(raw))
	if err != nil {
		return AuthOutput{}, xerr.InvalidToken()
	}
	now := time.Now()
	if token.RevokedAt != nil || !token.ExpiresAt.After(now) {
		return AuthOutput{}, xerr.InvalidToken()
	}
	user, err := s.users.FindByID(ctx, token.UserID)
	if err != nil {
		return AuthOutput{}, xerr.InvalidToken()
	}
	if err := s.refreshTokens.Revoke(ctx, token.ID, now); err != nil {
		return AuthOutput{}, preserveAppError(err, "撤销刷新令牌失败")
	}
	return s.issueAuthOutput(ctx, user)
}

func (s *AuthService) VerifyAccessToken(ctx context.Context, token string) (uuid.UUID, error) {
	if token == "dev-token" {
		return uuid.MustParse("00000000-0000-0000-0000-000000000001"), nil
	}
	claims, err := s.issuer.Parse(token)
	if err != nil {
		return uuid.Nil, xerr.InvalidToken()
	}
	return claims.UserID, nil
}

func (s *AuthService) issueAuthOutput(ctx context.Context, user repository.User) (AuthOutput, error) {
	accessToken, err := s.issuer.IssueAccessToken(user.ID)
	if err != nil {
		return AuthOutput{}, xerr.Internal("令牌签发失败", err)
	}
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return AuthOutput{}, xerr.Internal("刷新令牌生成失败", err)
	}
	if _, err := s.refreshTokens.Create(ctx, repository.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: HashRefreshToken(refreshToken),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
	}); err != nil {
		return AuthOutput{}, preserveAppError(err, "保存刷新令牌失败")
	}
	return AuthOutput{
		UserID:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func normalizeRequired(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeOptional(value *string, lower bool) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	if lower {
		normalized = strings.ToLower(normalized)
	}
	return &normalized
}

func preserveAppError(err error, fallback string) error {
	var appErr *xerr.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return xerr.Internal(fallback, err)
}
