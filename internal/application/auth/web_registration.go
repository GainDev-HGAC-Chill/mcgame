package auth

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	pkgerrors "mcgame-server/pkg/errors"
)

type WebRegistrationRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

func (r *WebRegistrationRequest) Validate() error {
	if r == nil || r.Username == "" || !utf8.ValidString(r.Username) || strings.ContainsRune(r.Username, 0) || len(utf16.Encode([]rune(r.Username))) > 50 {
		return pkgerrors.ErrInvalidInput
	}
	if r.Password == "" || !utf8.ValidString(r.Password) || len(utf16.Encode([]rune(r.Password))) > 20 || r.Password != r.ConfirmPassword {
		return pkgerrors.ErrInvalidInput
	}
	return nil
}

func (s *Service) RegisterWeb(ctx context.Context, req *WebRegistrationRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}
	digest := md5.Sum([]byte(req.Password))
	return s.Register(ctx, &RegisterRequest{
		Username: req.Username,
		Password: hex.EncodeToString(digest[:]),
	})
}
