package fetch

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"pxs/internal/config"
)

var userAgent = fmt.Sprintf("pxs/%s (ZulfaNurhuda; pxnotpixel; Seleksi Asisten Lab Basis Data)", config.UserAgentVersion)

var client = &http.Client{Timeout: config.HTTPTimeout}

// botChallengeMarker: halaman bot-challenge Fastly tetap balas HTTP 200, jadi tidak bisa dideteksi lewat status code saja.
const botChallengeMarker = "<title>Client Challenge</title>"

func Get(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("membuat request ke %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mengambil %s gagal: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mengambil %s gagal: status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("membaca response %s: %w", url, err)
	}

	if bytes.Contains(body, []byte(botChallengeMarker)) {
		return nil, fmt.Errorf("mengambil %s gagal: server menyajikan halaman bot-challenge, bukan konten asli", url)
	}

	return body, nil
}
