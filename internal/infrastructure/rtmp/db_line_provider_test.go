package rtmp

import "testing"

func TestPublicLineURL(t *testing.T) {
	for _, tc := range []struct {
		value, host, want string
	}{
		{"rtmp://127.0.0.1:1935/line/0", "26.31.111.123", "rtmp://26.31.111.123:1935/line/0"},
		{"rtmpe://localhost:1935/line/2?test=1", "26.31.111.123", "rtmpe://26.31.111.123:1935/line/2?test=1"},
		{"rtmp://127.0.0.1:1935/line/0", "192.168.2.9", "rtmp://192.168.2.9:1935/line/0"},
		{"rtmp://127.0.0.1:1935/line/0", "127.0.0.1", "rtmp://127.0.0.1:1935/line/0"},
		{"rtmp://127.0.0.1:1935/line/0", "", "rtmp://127.0.0.1:1935/line/0"},
		{"rtmp://127.0.0.1:1935/line/0", "invalid", "rtmp://127.0.0.1:1935/line/0"},
		{"rtmp://192.168.1.10:1935/line/0", "26.31.111.123", "rtmp://192.168.1.10:1935/line/0"},
		{"rtmpe://game.example.com:1935/line/0", "26.31.111.123", "rtmpe://game.example.com:1935/line/0"},
		{"rtmp://localhost/line/0", "26.31.111.123", "rtmp://26.31.111.123/line/0"},
		{"rtmp://[::1]:1935/line/0", "26.31.111.123", "rtmp://26.31.111.123:1935/line/0"},
		{"rtmp://localhost/line/0", "::1", "rtmp://[::1]/line/0"},
		{"http://localhost/line/0", "26.31.111.123", "http://localhost/line/0"},
		{"invalid", "26.31.111.123", "invalid"},
	} {
		if got := publicLineURL(tc.value, tc.host); got != tc.want {
			t.Fatalf("%+v: got %q", tc, got)
		}
	}
}
