package modmake

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"
)

var (
	ErrNoGoVersion = errors.New("go version not found")

	_goVersionMux   sync.RWMutex
	_goVersionCache map[int]string
)

// PinLatestV1 will replace the cached [GoTools] instance with one pinned to the latest patch version of the specified minor version.
// To revert back to the system go version, use [GoTools.InvalidateCache].
//
// Note: For safety reasons, unstable release candidate versions are not considered.
// If there is not a stable version available, then this function will panic.
func (g *GoTools) PinLatestV1(minorVersion int) *GoTools {
	Go().InvalidateCache()
	curSysInstance := Go()
	curGoBinPath := curSysInstance.GOBIN()
	// Get the latest patch version to pin to
	version, err := queryLatestPatch(queryGoVersions, minorVersion)
	if err != nil {
		panic(err)
	}
	if len(version) == 0 {
		panic(fmt.Errorf("%w: version 1.%d", ErrNoGoVersion, minorVersion))
	}
	// Install it
	if err := curSysInstance.Install(fmt.Sprintf("golang.org/dl/%s@latest", version)).Run(context.Background()); err != nil {
		panic(err)
	}
	// Get a direct reference to the executable
	pinnedGo := curGoBinPath.Join(version)
	// Download the GOROOT equivalent
	if err := Exec(pinnedGo.String(), "download").LogGroup("go download").Run(context.Background()); err != nil {
		panic(err)
	}
	// Initialize the new GoTools instance
	_goInstance = nil
	pinnedInstance, err := initGoInstNamed(pinnedGo.String())
	if err != nil {
		panic(fmt.Errorf("failed to initialize pinned go version '%s': %w", version, err))
	}
	_goInstance = pinnedInstance
	return pinnedInstance
}

type goVersionEntry struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func queryLatestPatch(queryGoVersions func() ([]goVersionEntry, error), minorVersion int) (string, error) {
	if _goVersionCache != nil {
		_goVersionMux.RLock()
		defer _goVersionMux.RUnlock()
		return _goVersionCache[minorVersion], nil
	}
	versions, err := queryGoVersions()
	if err != nil {
		return "", fmt.Errorf("failed to query go versions: %w", err)
	}

	var (
		maxCache = map[int]int{}
		matcher  = regexp.MustCompile(`^go1\.(\d+)\.(\d+)$`)
	)
	_goVersionMux.Lock()
	defer _goVersionMux.Unlock()
	for _, version := range versions {
		if !version.Stable {
			continue
		}
		if !matcher.MatchString(version.Version) {
			continue
		}
		groups := matcher.FindStringSubmatch(version.Version)
		minor, err := strconv.Atoi(groups[1])
		if err != nil || minor <= 0 {
			continue
		}
		patch, err := strconv.Atoi(groups[2])
		if err != nil {
			continue
		}
		if maxCache[minor] <= patch {
			maxCache[minor] = patch
		}
	}
	_goVersionCache = map[int]string{}
	for minor, patch := range maxCache {
		_goVersionCache[minor] = fmt.Sprintf("go1.%d.%d", minor, patch)
	}
	return _goVersionCache[minorVersion], nil
}

func queryGoVersions() ([]goVersionEntry, error) {
	req, err := http.NewRequest(http.MethodGet, "https://go.dev/dl/", nil)
	if err != nil {
		return nil, err
	}
	queryVals := req.URL.Query()
	queryVals.Set("mode", "json")
	queryVals.Set("include", "all")
	req.URL.RawQuery = queryVals.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status '%s' from go.dev", resp.Status)
	}
	var versions []goVersionEntry
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to parse go.dev versions: %w", err)
	}
	return versions, nil
}
