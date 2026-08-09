package cmd

import "testing"

func TestDownloadLooksBinary(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		prefix      []byte
		want        bool
	}{
		{name: "octet stream", contentType: "application/octet-stream", prefix: []byte("plain"), want: true},
		{name: "image", contentType: "image/png", prefix: []byte("not enough"), want: true},
		{name: "text", contentType: "text/plain; charset=utf-8", prefix: []byte{0x00, 0x01}, want: false},
		{name: "json", contentType: "application/json", prefix: []byte(`{"ok":true}`), want: false},
		{name: "xml", contentType: "application/xml", prefix: []byte("<ok/>"), want: false},
		{name: "javascript", contentType: "application/javascript", prefix: []byte("const ok = true;"), want: false},
		{name: "yaml", contentType: "application/yaml", prefix: []byte("ok: true\n"), want: false},
		{name: "x-yaml", contentType: "application/x-yaml", prefix: []byte("ok: true\n"), want: false},
		{name: "yaml charset", contentType: "application/yaml; charset=utf-8", prefix: []byte("ok: true\n"), want: false},
		{name: "text yaml", contentType: "text/yaml", prefix: []byte("ok: true\n"), want: false},
		{name: "unsupported yaml alias", contentType: "application/x-yml", prefix: []byte("ok: true\n"), want: true},
		{name: "detected binary", prefix: []byte{0x89, 'P', 'N', 'G'}, want: true},
		{name: "detected text", prefix: []byte("release notes\n"), want: false},
		{name: "invalid utf8", prefix: []byte{0xff, 0xfe}, want: true},
		{name: "null byte", prefix: []byte{0x00, 'x'}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := downloadLooksBinary(test.contentType, test.prefix); got != test.want {
				t.Fatalf("downloadLooksBinary(%q, %v) = %v, want %v", test.contentType, test.prefix, got, test.want)
			}
		})
	}
}
