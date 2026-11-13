package httpx

import (
	"encoding/json"
	"net/url"
	"testing"
)

func FuzzRedactOAuthForm(f *testing.F) {
	f.Add("grant_type=refresh_token&refresh_token=abc&client_id=c")
	f.Add("code=x&code_verifier=y")
	f.Add("refresh%5Ftoken=abc")
	f.Fuzz(func(t *testing.T, body string) {
		out := redactOAuthForm([]byte(body), "application/x-www-form-urlencoded")
		q, err := url.ParseQuery(string(out))
		if err != nil {
			return
		}
		for k, vs := range q {
			if !sensitiveFormKey(k) {
				continue
			}
			for _, v := range vs {
				if v != "<redacted>" && v != "" {
					t.Fatalf("%s=%q survived redaction of %q", k, v, body)
				}
			}
		}
	})
}

func FuzzRedactJSONTokens(f *testing.F) {
	f.Add(`{"access_token":"a","nested":[{"token":"b"}]}`)
	f.Add(`{"client_secret":"s"}`)
	f.Fuzz(func(t *testing.T, body string) {
		out := redactJSONTokens([]byte(body))
		var v any
		if json.Unmarshal(out, &v) != nil {
			return
		}
		var walk func(any)
		walk = func(v any) {
			switch x := v.(type) {
			case map[string]any:
				for k, child := range x {
					if sensitiveTokenKey(k) {
						if s, ok := child.(string); !ok || s != "<redacted>" {
							t.Fatalf("%s=%v survived redaction of %q", k, child, body)
						}
						continue
					}
					walk(child)
				}
			case []any:
				for _, child := range x {
					walk(child)
				}
			}
		}
		walk(v)
	})
}
