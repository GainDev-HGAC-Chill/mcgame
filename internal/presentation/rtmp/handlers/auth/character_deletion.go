package auth

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

// DeleteNow must validate account ownership and delete the character atomically
// inside PostgreSQL. The game must never trust a client-supplied account ID.
type CharacterDeletionStore interface {
	DeleteNow(context.Context, uuid.UUID, int64) (bool, error)
}

type CharacterDeletionCache interface {
	InvalidateCharacter(context.Context, int64) error
}

func (h *Handler) SetCharacterDeletionStore(store CharacterDeletionStore) {
	h.deletionStore = store
}

func (h *Handler) SetCharacterDeletionCache(cache CharacterDeletionCache) {
	h.deletionCache = cache
}

func (h *Handler) deletionAccountID(ctx *rtmp.RPCContext) (uuid.UUID, error) {
	if ctx == nil || ctx.Connection == nil || ctx.Connection.IsGlobalConnection() || ctx.AccountID == "" || ctx.CharacterID != "" {
		return uuid.Nil, pkgerrors.ErrUnauthorized
	}
	accountID, err := uuid.Parse(ctx.AccountID)
	if err != nil {
		return uuid.Nil, pkgerrors.ErrUnauthorized
	}
	return accountID, nil
}

// StartDeleteChar keeps the Flash RPC contract (character ID, secondary-password MD5).
// Unlike the old 7-day implementation, successful authentication deletes immediately.
func (h *Handler) StartDeleteChar(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if h.deletionStore == nil || h.charService == nil || h.accountRepo == nil {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	if len(args) != 2 {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidArgs), nil
	}
	characterID, ok := parseInt64Arg(args[0])
	if !ok || characterID <= 0 {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidArgs), nil
	}
	passwordMD5, ok := args[1].(string)
	if !ok || !validDeletionMD5(passwordMD5) {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	accountID, err := h.deletionAccountID(ctx)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	account, err := h.accountRepo.FindByID(ctx.Context, accountID)
	if err != nil || account == nil {
		return rtmp.ErrorToResponse(pkgerrors.ErrUnauthorized), nil
	}
	if !account.CheckSecondaryPassword(strings.ToLower(passwordMD5)) {
		h.logger.Warn("StartDeleteChar: incorrect secondary password", zap.String("account_id", ctx.AccountID))
		h.sendCharacterListCallback(ctx)
		return rtmp.ErrorToResponse(pkgerrors.ErrUnauthorized), nil
	}
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil || char == nil || char.AccountID != accountID {
		return rtmp.ErrorToResponse(pkgerrors.ErrCharacterNotOwned), nil
	}
	if h.rtmpServer != nil && h.rtmpServer.IsCharacterOnline(strconv.FormatInt(characterID, 10)) {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	deleted, err := h.deletionStore.DeleteNow(ctx.Context, accountID, characterID)
	if err != nil {
		h.logger.Error("StartDeleteChar: immediate deletion failed", zap.Int64("character_id", characterID), zap.Error(err))
		return rtmp.ErrorToResponse(err), nil
	}
	if !deleted {
		return rtmp.ErrorToResponse(pkgerrors.ErrCharacterNotOwned), nil
	}
	// DB deletion has committed; never report it as failed merely because Redis
	// invalidation failed. Log the latter so operators can investigate.
	if h.deletionCache != nil {
		if cacheErr := h.deletionCache.InvalidateCharacter(ctx.Context, characterID); cacheErr != nil {
			h.logger.Error("StartDeleteChar: cache invalidation failed after delete", zap.Int64("character_id", characterID), zap.Error(cacheErr))
		}
	}
	h.logger.Info("StartDeleteChar: character permanently deleted", zap.String("account_id", ctx.AccountID), zap.Int64("character_id", characterID))
	h.sendCharacterListCallback(ctx)
	return nil, nil
}

func validDeletionMD5(s string) bool {
	if len(s) != 32 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
