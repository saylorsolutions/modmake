package modmake

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func Download(url string, location PathString) Task {
	url = strings.TrimSpace(url)
	if len(url) == 0 {
		panic("empty URL")
	}
	if len(location) == 0 {
		panic("empty download location")
	}
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send HTTP request: %v", err)
		}
		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()
		if resp.StatusCode != 200 {
			return fmt.Errorf("expected status 200 OK, got %s", resp.Status)
		}

		out, err := os.Create(location.String())
		if err != nil {
			return err
		}
		defer func() {
			_ = out.Close()
		}()
		_, err = io.Copy(out, resp.Body)
		return err
	}
}
