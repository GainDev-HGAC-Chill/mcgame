// Absolute level changes use the canonical domain transition and cache-aware mutation.
package adminhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"go.uber.org/zap"

	"mcgame-server/internal/domain/character"
	"mcgame-server/internal/domain/item"
)

type ProgressionStatProvider interface {
	AggregateEquipmentStats(context.Context, int64) character.EquipmentStatBonuses
}

type ConditionalCharacterUpdater interface {
	UpdateCachedCharacterConditional(int64, func(*character.Character, []*item.Item) (bool, error)) (*character.Character, bool, error)
}

func (s *Server) SetProgressionStatProvider(provider ProgressionStatProvider) {
	s.progressionStats = provider
}

type targetLevelPayload struct {
	TargetLevel *int `json:"targetLevel"`
}

func parseTargetLevel(raw json.RawMessage) (int, error) {
	var payload targetLevelPayload
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return 0, fmt.Errorf("cấp đích phải là số nguyên")
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return 0, fmt.Errorf("dữ liệu cấp đích không hợp lệ")
	}
	if payload.TargetLevel == nil || *payload.TargetLevel < 1 || *payload.TargetLevel > character.MaxLevel {
		return 0, fmt.Errorf("cấp đích phải từ 1 đến %d", character.MaxLevel)
	}
	return *payload.TargetLevel, nil
}

func (s *Server) executeSetTargetLevel(w http.ResponseWriter, ctx context.Context, char *character.Character, raw json.RawMessage) {
	target, err := parseTargetLevel(raw)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, adminErrorResponse{Message: err.Error()})
		return
	}
	if s.templateProvider == nil || s.progressionStats == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, adminErrorResponse{Message: "Chưa cấu hình kiểm tra trang bị và tính thuộc tính."})
		return
	}
	updater, ok := s.characterCacheUpdater.(ConditionalCharacterUpdater)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, adminErrorResponse{Message: "Chưa cấu hình chuyển cấp có khóa an toàn; chưa thay đổi cấp."})
		return
	}
	var refunded int
	apply := func(current *character.Character, items []*item.Item) (bool, error) {
		if target < current.Level {
			if err := s.validateTargetEquipment(items, target); err != nil {
				return false, err
			}
			refunded = current.DistributedAttrPoints
		}
		return current.SetTargetLevel(target)
	}
	updated, changed, mutationErr := updater.UpdateCachedCharacterConditional(char.ID, apply)
	delivery := "cached_state"
	if updated == nil {
		s.writeJSON(w, http.StatusConflict, adminErrorResponse{Message: "Nhân vật chưa có dữ liệu bộ nhớ đệm đang hoạt động. Thay đổi ngoại tuyến chưa được hỗ trợ an toàn; chưa thay đổi cấp."})
		return
	}
	if mutationErr != nil {
		s.writeJSON(w, http.StatusBadRequest, adminErrorResponse{Message: mutationErr.Error()})
		return
	}
	if !changed {
		s.writeTargetLevelResult(w, updated, "no_change", false, 0)
		return
	}
	conn := s.getConnection(char.ID)
	if conn != nil {
		bonuses := s.progressionStats.AggregateEquipmentStats(ctx, char.ID)
		payload := character.BuildStatRefreshUPPPayloadWithEquipment(updated, bonuses)
		if err := conn.SendCallback("onUPP", payload); err != nil {
			s.logger.Warn("admin target level callback failed", zap.Int64("character_id", char.ID), zap.Error(err))
			delivery = "live_session_callback_failed"
		} else {
			delivery = "live_session"
		}
	}
	s.writeTargetLevelResult(w, updated, delivery, true, refunded)
}

func (s *Server) validateTargetEquipment(equipped []*item.Item, target int) error {
	for _, it := range equipped {
		if it == nil || it.SlotType != item.SlotTypeEquipped {
			continue
		}
		tpl := s.templateProvider.GetEquipmentTemplate(it.TemplateID)
		if tpl == nil || math.IsNaN(tpl.ReqLevel) || math.IsInf(tpl.ReqLevel, 0) {
			return fmt.Errorf("không thể xác minh trang bị %d; chưa hạ cấp", it.ID)
		}
		if float64(target) < tpl.ReqLevel {
			return fmt.Errorf("trang bị %s (ID %d) cần cấp %.0f; tháo trang bị trước khi hạ xuống cấp %d", tpl.Name, it.ID, tpl.ReqLevel, target)
		}
	}
	return nil
}

func (s *Server) writeTargetLevelResult(w http.ResponseWriter, char *character.Character, delivery string, changed bool, refunded int) {
	message := "Đã thay đổi cấp. Tài sản, lịch sử và kỹ năng được giữ nguyên."
	if !changed {
		message = "Cấp hiện tại đã bằng cấp đích; không thay đổi dữ liệu."
	} else {
		message += " Dữ liệu bộ nhớ đệm sẽ được lưu theo cơ chế hiện hành; phản hồi này không xác nhận đã ghi xuống cơ sở dữ liệu."
	}
	s.writeJSON(w, http.StatusOK, adminActionResponse{
		OK: true, Action: "set_target_level", TargetID: char.ID, DeliveryMode: delivery, StatusMessage: message,
		Data: map[string]interface{}{
			"changed": changed, "level": char.Level, "experience": char.Experience,
			"attrPoints": char.AttrPoints, "maxAttrPoints": char.MaxAttrPoints,
			"distributedAttrPoints": char.DistributedAttrPoints, "refundedAttrPoints": refunded,
		},
	})
}
