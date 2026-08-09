package cmd

import "testing"

func TestReleaseAssetLooksBinary(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		prefix      []byte
		want        bool
	}{
		{name: "octet stream", contentType: "application/octet-stream", prefix: []byte("plain"), want: true},
		{name: "image", contentType: "image/png", prefix: []byte("not enough"), want: true},
		{name: "text", contentType: "text/plain; charset=utf-8", prefix: []byte{0x00, 0x01}, want: false},
		{name: "detected binary", prefix: []byte{0x89, 'P', 'N', 'G'}, want: true},
		{name: "detected text", prefix: []byte("release notes\n"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := releaseAssetLooksBinary(test.contentType, test.prefix); got != test.want {
				t.Fatalf("releaseAssetLooksBinary(%q, %v) = %v, want %v", test.contentType, test.prefix, got, test.want)
			}
		})
	}
}
