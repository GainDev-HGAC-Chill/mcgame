package registration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	appauth "mcgame-server/internal/application/auth"
	pkgerrors "mcgame-server/pkg/errors"
)

type Registerer interface {
	RegisterWeb(context.Context, *appauth.WebRegistrationRequest) error
}

type Handler struct {
	registerer Registerer
	permits    chan struct{}
}

func NewHandler(registerer Registerer) *Handler {
	return &Handler{registerer: registerer, permits: make(chan struct{}, 4)}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.URL.Path != "/api/register" || r.URL.EscapedPath() != "/api/register" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		respond(w, http.StatusMethodNotAllowed, "method_not_allowed", "Chỉ hỗ trợ POST.")
		return
	}
	if !sameOrigin(r) {
		respond(w, http.StatusBadRequest, "invalid_origin", "Vui lòng đăng ký từ trang cùng máy chủ.")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		respond(w, http.StatusBadRequest, "invalid_input", "Yêu cầu phải là JSON.")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil || !validJSONUnicode(body) {
		respond(w, http.StatusBadRequest, "invalid_input", "Dữ liệu không hợp lệ hoặc vượt quá 4 KiB.")
		return
	}
	req, err := decodeRequest(body)
	if err != nil || req.Validate() != nil {
		respond(w, http.StatusBadRequest, "invalid_input", "Điền đầy đủ thông tin: ID tối đa 50 đơn vị UTF-16, mật khẩu tối đa 20 đơn vị UTF-16 và xác nhận phải trùng khớp.")
		return
	}
	select {
	case h.permits <- struct{}{}:
		defer func() { <-h.permits }()
	default:
		w.Header().Set("Retry-After", "12")
		respond(w, http.StatusTooManyRequests, "rate_limited", "Máy chủ đang bận. Vui lòng chờ rồi thử lại.")
		return
	}
	if h.registerer == nil {
		respond(w, http.StatusServiceUnavailable, "unavailable", "Chưa thể đăng ký. Vui lòng thử lại sau.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), registrationTimeout)
	defer cancel()
	err = h.registerer.RegisterWeb(ctx, req)
	switch {
	case err == nil:
		respond(w, http.StatusCreated, "created", "Đăng ký thành công! Quay lại game Flash để đăng nhập và tạo nhân vật.")
	case errors.Is(err, pkgerrors.ErrAlreadyExists):
		respond(w, http.StatusConflict, "username_taken", "ID này đã tồn tại. Vui lòng chọn ID khác.")
	case errors.Is(err, pkgerrors.ErrInvalidInput):
		respond(w, http.StatusBadRequest, "invalid_input", "Thông tin đăng ký không hợp lệ.")
	default:
		respond(w, http.StatusServiceUnavailable, "unavailable", "Chưa thể đăng ký. Vui lòng thử lại sau.")
	}
}

func sameOrigin(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}
	origins := r.Header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if len(origins) != 1 {
		return false
	}
	origin, err := url.Parse(origins[0])
	if err != nil || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return origin.Scheme == scheme && strings.EqualFold(origin.Host, r.Host)
}

func respond(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{code, message})
}
