package auth

import (
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"mcgame-server/internal/infrastructure/rtmp"
	pkgerrors "mcgame-server/pkg/errors"
)

const characterDeletionDelay = 7 * 24 * time.Hour

type CharacterDeletionStore interface {
	ScheduleDeletion(context.Context, uuid.UUID, int64, time.Time) (bool, error)
	RestoreDeletion(context.Context, uuid.UUID, int64) (bool, error)
	CharacterDeletionTimes(context.Context, uuid.UUID) (map[int64]int64, error)
	IsDeletionPending(context.Context, uuid.UUID, int64) (bool, error)
	PurgeOneExpired(context.Context, uuid.UUID, int64) (bool, error)
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

func (h *Handler) SetCharacterDeletionPurgeEnabled(enabled bool) {
	h.deletionPurgeEnabled = enabled
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
	applied, err := h.deletionStore.ScheduleDeletion(ctx.Context, accountID, characterID, time.Now().Add(characterDeletionDelay))
	if err != nil {
		h.logger.Error("StartDeleteChar: scheduling failed", zap.Error(err), zap.Int64("character_id", characterID))
		return rtmp.ErrorToResponse(err), nil
	}
	h.logger.Info("StartDeleteChar processed", zap.Int64("character_id", characterID), zap.Bool("created", applied))
	h.sendCharacterListCallback(ctx)
	return nil, nil
}

func (h *Handler) RestoreChar(ctx *rtmp.RPCContext, args []interface{}) (interface{}, error) {
	if h.deletionStore == nil || h.charService == nil {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidInput), nil
	}
	if len(args) < 1 || len(args) > 2 {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidArgs), nil
	}
	characterID, ok := lastInt64Arg(args)
	if !ok || characterID <= 0 {
		return rtmp.ErrorToResponse(pkgerrors.ErrInvalidArgs), nil
	}
	accountID, err := h.deletionAccountID(ctx)
	if err != nil {
		return rtmp.ErrorToResponse(err), nil
	}
	char, err := h.charService.GetByID(ctx.Context, characterID)
	if err != nil || char == nil || char.AccountID != accountID {
		return rtmp.ErrorToResponse(pkgerrors.ErrCharacterNotOwned), nil
	}
	restored, err := h.deletionStore.RestoreDeletion(ctx.Context, accountID, characterID)
	if err != nil {
		h.logger.Error("RestoreChar failed", zap.Error(err), zap.Int64("character_id", characterID))
		return rtmp.ErrorToResponse(err), nil
	}
	h.logger.Info("RestoreChar processed", zap.Int64("character_id", characterID), zap.Bool("restored", restored))
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

func (h *Handler) deletionTimesForAccount(ctx *rtmp.RPCContext, accountID uuid.UUID) (map[int64]int64, error) {
	if h.deletionStore == nil {
		return nil, pkgerrors.ErrInvalidInput
	}
	times, err := h.deletionStore.CharacterDeletionTimes(ctx.Context, accountID)
	if err != nil || !h.deletionPurgeEnabled || h.deletionCache == nil {
		return times, err
	}
	now := time.Now().Unix()
	purged := false
	for id, deleteAt := range times {
		if deleteAt > now || (h.rtmpServer != nil && h.rtmpServer.IsCharacterOnline(strconv.FormatInt(id, 10))) {
			continue
		}
		ok, purgeErr := h.deletionStore.PurgeOneExpired(ctx.Context, accountID, id)
		if purgeErr != nil {
			h.logger.Error("Failed to purge expired character", zap.Int64("character_id", id), zap.Error(purgeErr))
			continue
		}
		if ok {
			purged = true
			if invalidateErr := h.deletionCache.InvalidateCharacter(ctx.Context, id); invalidateErr != nil {
				h.logger.Error("Failed to invalidate deleted character cache", zap.Int64("character_id", id), zap.Error(invalidateErr))
			}
		}
	}
	if purged {
		return h.deletionStore.CharacterDeletionTimes(ctx.Context, accountID)
	}
	return times, nil
}
