package registration

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	appauth "mcgame-server/internal/application/auth"
	pkgerrors "mcgame-server/pkg/errors"
)

const registrationTimeout = 10 * time.Second

func decodeRequest(body []byte) (*appauth.WebRegistrationRequest, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, pkgerrors.ErrInvalidInput
	}
	fields := make(map[string]string, 3)
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, pkgerrors.ErrInvalidInput
		}
		key, ok := token.(string)
		if !ok || (key != "username" && key != "password" && key != "confirmPassword") {
			return nil, pkgerrors.ErrInvalidInput
		}
		if _, exists := fields[key]; exists {
			return nil, pkgerrors.ErrInvalidInput
		}
		token, err = decoder.Token()
		value, ok := token.(string)
		if err != nil || !ok {
			return nil, pkgerrors.ErrInvalidInput
		}
		fields[key] = value
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') || len(fields) != 3 {
		return nil, pkgerrors.ErrInvalidInput
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, pkgerrors.ErrInvalidInput
	}
	return &appauth.WebRegistrationRequest{Username: fields["username"], Password: fields["password"], ConfirmPassword: fields["confirmPassword"]}, nil
}

func validJSONUnicode(body []byte) bool {
	if !utf8.Valid(body) || !json.Valid(body) {
		return false
	}
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' {
			continue
		}
		i++
		if i >= len(body) || body[i] != 'u' {
			continue
		}
		value, err := strconv.ParseUint(string(body[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if value >= 0xdc00 && value <= 0xdfff {
			return false
		}
		if value < 0xd800 || value > 0xdbff {
			continue
		}
		if i+6 >= len(body) || body[i+1] != '\\' || body[i+2] != 'u' {
			return false
		}
		low, err := strconv.ParseUint(string(body[i+3:i+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}
