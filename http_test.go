package modmake

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestDownload(t *testing.T) {
	location := filepath.Join("build", "index.html")
	err := Script(
		IfExists(location, WithoutContext(func() error {
			t.Errorf("File '%[1]s' existed before starting, delete '%[1]s' and run this test again to get past the pre-check", location)
			return errors.New("precondition not met")
		})),
		Mkdir("build", 0755),
		Download("https://google.com", location).
			Catch(func(err error) error {
				t.Error("Unexpected error returned:", err)
				return err
			}),
		IfNotExists(location, Plain(func() {
			t.Error("Download file should exist now")
		})),
		RemoveFile(location),
	).Run(context.Background())
	assert.NoError(t, err)
}
