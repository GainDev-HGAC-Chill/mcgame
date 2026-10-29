// Open-sourced by BaoLT

package marriage

import (
	"context"
	"time"

	appmarriage "mcgame-server/internal/application/marriage"
	"mcgame-server/internal/infrastructure/rtmp"

	"go.uber.org/zap"
)

type Handler struct {
	service   *appmarriage.Service
	logger    *zap.Logger
	transport weddingTransport
}

type weddingTransport interface {
	TransportPlayerAlone(charID int64, mapID, x, y int) bool
}

func NewHandler(service *appmarriage.Service, logger *zap.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) SetWeddingTransport(transport weddingTransport) {
	h.transport = transport
}

func (h *Handler) SendWeddingBookList(ctx context.Context, connection *rtmp.Connection) error {
	bookings, err := h.service.WeddingBookList(ctx, time.Now())
	if err != nil {
		return err
	}
	return connection.SendCallback("onInitBookList", bookings)
}

func (h *Handler) RegisterHandlers(dispatcher *rtmp.RPCDispatcher) {
	dispatcher.Register("initMarriage", h.InitMarriage)
	dispatcher.Register("marriageSeeking", h.MarriageSeeking)
	dispatcher.Register("marriageRequest", h.MarriageRequest)
	dispatcher.Register("marriageReqFeedback", h.MarriageReqFeedback)
	dispatcher.Register("cancelMarriageSekInfo", h.CancelMarriageSekInfo)
	dispatcher.Register("getCoupleRank", h.GetCoupleRank)
	dispatcher.Register("bookWeddingHall", h.BookWeddingHall)
	dispatcher.Register("cancelWeddingHall", h.CancelWeddingHall)
	dispatcher.Register("enterWeddingHall", h.EnterWeddingHall)
}
