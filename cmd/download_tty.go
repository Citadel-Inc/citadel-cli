package cmd

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

var downloadOutputIsTTY = func(w io.Writer) bool {
	file, ok := w.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func downloadLooksBinary(contentType string, prefix []byte) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	if mediaType != "" {
		return !downloadTextMediaType(mediaType)
	}
	if bytes.IndexByte(prefix, 0) >= 0 || !utf8.Valid(prefix) {
		return true
	}
	return !downloadTextMediaType(strings.ToLower(http.DetectContentType(prefix)))
}

func downloadTextMediaType(mediaType string) bool {
	return strings.HasPrefix(mediaType, "text/") ||
		mediaType == "application/json" ||
		mediaType == "application/xml" ||
		mediaType == "application/javascript"
}
