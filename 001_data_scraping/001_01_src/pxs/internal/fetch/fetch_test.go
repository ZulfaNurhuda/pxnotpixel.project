package fetch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGet_SendsUserAgentAndReturnsBody(t *testing.T) {
	var gotUA string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "<html>hello</html>")
	}))
	defer server.Close()

	body, err := Get(server.URL)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(body) != "<html>hello</html>" {
		t.Fatalf("unexpected body: %q", body)
	}
	if !strings.HasPrefix(gotUA, "pxs/") {
		t.Fatalf("expected User-Agent to start with 'pxs/', got %q", gotUA)
	}
}

func TestGet_NonOKStatusIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Get(server.URL)
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

// TestGet_BotChallengePageIsError: tanpa cek ini, halaman bot-challenge (200 OK) diam-diam ke-parse jadi entitas nyaris kosong, bukan error fetch.
func TestGet_BotChallengePageIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "<html><head><title>Client Challenge</title></head><body></body></html>")
	}))
	defer server.Close()

	_, err := Get(server.URL)
	if err == nil {
		t.Fatal("expected error for a bot-challenge page (HTTP 200 but not real content), got nil")
	}
}
