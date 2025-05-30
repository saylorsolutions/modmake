package minify

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	mm "github.com/saylorsolutions/modmake"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"unicode"
)

const (
	minifyV2Path  = "github.com/tdewolff/minify/v2/cmd/minify@%s"
	EnvMinifyPath = "MM_MINIFY_PATH"
)

var (
	minifyVersionPattern = regexp.MustCompile(`^(latest|v2\.\d+\.\d+)$`)
	minifierPath         string
	minifierInitOnce     sync.Once
	stripSpacePattern    = regexp.MustCompile(`\s`)

	//go:embed mappingFile.got
	mappingTemplateText string
	mappingTemplate     = template.Must(template.New("mappingTemplate").Parse(mappingTemplateText))
)

// ConfigFunc is a function that is able to define values for Configuration.
type ConfigFunc func(conf *Configuration) error

type templateFile struct {
	Package         string
	MinifiedRelPath string
	EmbedSymbol     string
	FileName        string
}

// Configuration defines consistent policies for minifying web assets.
// The type is immutable after creation with NewConfig.
type Configuration struct {
	mappingFile       mm.PathString
	mappingFileHandle *os.File
	closeFile         sync.Once
	hashDigits        int
	minifyVersion     string
	tasks             mm.Task
	assetDir          mm.PathString
	clearBeforeWrite  bool
	packageName       string
}

// HashDigits sets the number of hash digits to use for minified files.
// Digits must be between 4-32, inclusive.
func HashDigits(digits int) ConfigFunc {
	return func(conf *Configuration) error {
		if digits < 4 {
			return errors.New("hash digits must be at least 4, recommend at least 6 (default)")
		}
		if digits > 32 {
			return errors.New("hash digits cannot be greater than 32")
		}
		conf.hashDigits = digits
		return nil
	}
}

// Version sets the version of minify to use.
// Defaults to "latest".
func Version(version string) ConfigFunc {
	return func(conf *Configuration) error {
		if !minifyVersionPattern.MatchString(version) {
			return fmt.Errorf("invalid version string '%s'", version)
		}
		conf.minifyVersion = version
		return nil
	}
}

// MappingFilePackage sets the package name used in the mapping file.
// The default when no package is specified is the parent directory name.
func MappingFilePackage(packageName string) ConfigFunc {
	return func(conf *Configuration) error {
		if !token.IsIdentifier(packageName) {
			return fmt.Errorf("package name '%s' is not valid", packageName)
		}
		conf.packageName = packageName
		return nil
	}
}

// ClearBeforeWrite will make this Configuration clear the asset directory before writing new files.
func ClearBeforeWrite() ConfigFunc {
	return func(conf *Configuration) error {
		conf.clearBeforeWrite = true
		return nil
	}
}

func installTask(version string) mm.Task {
	return func(ctx context.Context) error {
		ctx, _ = mm.WithGroup(ctx, "install-minify")
		return mm.Go().Install(fmt.Sprintf(minifyV2Path, version)).Run(ctx)
	}
}

func NewConfig(mappingFile mm.PathString, assetDirName string, configFuncs ...ConfigFunc) (*Configuration, error) {
	assetDir := mappingFile.Dir().Join(assetDirName)
	mappingFileStr := mappingFile.String()
	if !strings.HasSuffix(mappingFileStr, ".go") {
		mappingFileStr += ".go"
		mappingFile = mm.Path(mappingFileStr)
	}
	conf := &Configuration{
		mappingFile:   mappingFile,
		assetDir:      assetDir,
		hashDigits:    6,
		minifyVersion: "latest",
	}
	var errs []error
	for _, fn := range configFuncs {
		if err := fn(conf); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(conf.packageName) == 0 {
		dir, err := mappingFile.Dir().Abs()
		if err != nil {
			return nil, err
		}
		conf.packageName = dir.Base().String()
	}
	conf.tasks = installTask(conf.minifyVersion)
	if conf.clearBeforeWrite {
		conf.tasks = conf.tasks.Then(mm.WithoutContext(func() error {
			return assetDir.RemoveAll()
		}))
	}
	conf.tasks = conf.tasks.Then(mm.WithoutContext(func() error {
		err := assetDir.Mkdir(0700)
		if err != nil {
			return fmt.Errorf("failed to create asset directory '%s': %w", assetDir, err)
		}
		conf.mappingFileHandle, err = conf.mappingFile.Create()
		if err != nil {
			return fmt.Errorf("failed to create mapping file '%s': %w", conf.mappingFile, err)
		}
		err = mappingTemplate.ExecuteTemplate(conf.mappingFileHandle, "fileHeader", &templateFile{
			Package: conf.packageName,
		})
		if err != nil {
			conf.closeFile.Do(func() {
				_ = conf.mappingFileHandle.Close()
				conf.mappingFileHandle = nil
			})
			return fmt.Errorf("failed to write mapping file header: %w", err)
		}
		return nil
	}))
	minifierInitOnce.Do(func() {
		defaultPath := mm.Path(mm.Go().GetEnv("GOBIN"), "minify").String()
		minifierPath = mm.F(fmt.Sprintf("${%s:%s}", EnvMinifyPath, defaultPath))
	})
	return conf, nil
}

func (conf *Configuration) args() []string {
	return []string{
		"--all",
		"--js-version", "0",
		"--html-keep-document-tags",
		"--html-keep-end-tags",
		"--html-keep-quotes",
	}
}

func embedSymbol(source mm.PathString) (string, error) {
	base := source.Base().String()
	base = stripSpacePattern.ReplaceAllString(base, "")
	var buf strings.Builder
	baseRunes := []rune(base)
	readFirst := false
	for _, r := range baseRunes {
		if !readFirst {
			if !unicode.IsLetter(r) {
				continue
			}
			buf.WriteRune(unicode.ToUpper(r))
			readFirst = true
			continue
		}
		switch {
		case unicode.IsLetter(r):
			fallthrough
		case unicode.IsNumber(r):
			fallthrough
		case r == '_':
			buf.WriteRune(r)
		case r == '-':
			buf.WriteRune('_')
		}
	}
	id := buf.String()
	if len(id) == 0 || !token.IsIdentifier(id) {
		return "", fmt.Errorf("unable to derive embed symbol from file name '%s'", source)
	}
	return id, nil
}

func (conf *Configuration) invokeMinify(source mm.PathString) mm.Task {
	return func(ctx context.Context) error {
		if conf.mappingFileHandle == nil {
			panic("unexpected state: mappingFileHandle invalid")
		}
		if !source.IsFile() {
			return fmt.Errorf("source file '%s' doesn't exist", source)
		}
		if !conf.assetDir.IsDir() {
			return fmt.Errorf("asset directory '%s' isn't valid", conf.assetDir)
		}
		fi, err := source.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat source file '%s': %w", source, err)
		}
		origSize := float64(fi.Size())
		var (
			buf bytes.Buffer
		)
		srcStr := source.String()
		ctx, log := mm.WithGroup(ctx, "minify "+srcStr)
		err = conf.getMinifiedContent(ctx, &buf, source)
		if err != nil {
			return err
		}
		minifiedSize := float64(buf.Len())
		if minifiedSize == 0 {
			return fmt.Errorf("no minified data written for file '%s'", source)
		}
		hash, err := hashFile(conf.hashDigits, source)
		if err != nil {
			return fmt.Errorf("failed to hash file '%s': %w", source, err)
		}
		targetFileName := hashedFileName(source, hash)
		targetFilePath := conf.assetDir.Join(targetFileName)
		embedIdentifier, err := embedSymbol(source)
		if err != nil {
			return err
		}
		relTarget, err := conf.mappingFile.Dir().Rel(targetFilePath)
		if err != nil {
			return fmt.Errorf("failed to get path to source '%s' relative to mapping file '%s': %w", source, conf.mappingFile, err)
		}
		err = targetFilePath.WriteFile(buf.Bytes(), 0600)
		if err != nil {
			return err
		}

		err = mappingTemplate.ExecuteTemplate(conf.mappingFileHandle, "fileMapping", templateFile{
			MinifiedRelPath: relTarget.ToSlash(),
			EmbedSymbol:     embedIdentifier,
			FileName:        targetFileName,
		})
		err = mappingTemplate.ExecuteTemplate(conf.mappingFileHandle, "fileMapping", templateFile{
			MinifiedRelPath: relTarget.ToSlash(),
			EmbedSymbol:     embedIdentifier,
			FileName:        targetFileName,
		})
		if err != nil {
			return fmt.Errorf("failed to write embed definition to mapping file: %w", err)
		}

		reduction := (origSize - minifiedSize) / origSize * 100.0
		log.Info("File '%s' reduced by %0.2f%% and written to '%s'\n", srcStr, reduction, targetFilePath)
		return nil
	}
}

func (conf *Configuration) getMinifiedContent(ctx context.Context, buf *bytes.Buffer, source mm.PathString) error {
	err := mm.Exec(minifierPath).Stdout(buf).
		Arg(conf.args()...).
		TrailingArg(source.String()).
		LogGroup("minify").
		Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to invoke minify: %w", err)
	}
	return nil
}

func (conf *Configuration) Run(ctx context.Context) error {
	return conf.tasks.Finally(func(terr error) error {
		var err error
		conf.closeFile.Do(func() {
			if conf.mappingFileHandle != nil {
				err = errors.Join(terr, conf.mappingFileHandle.Close())
				conf.mappingFileHandle = nil
			}
		})
		return err
	}).Run(ctx)
}

func hashedFileName(source mm.PathString, hash string) string {
	base := source.Base().String()
	ext := filepath.Ext(base)
	if len(ext) == 0 {
		return fmt.Sprintf("%s_%s", base, hash)
	}
	var buf strings.Builder
	extIdx := strings.LastIndex(base, ext)
	buf.Write([]byte(base)[:extIdx])
	buf.WriteString("." + hash)
	buf.WriteString(ext)
	return buf.String()
}

func hashFile(digits int, path mm.PathString) (string, error) {
	input, err := path.Open()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = input.Close()
	}()
	var hash []byte
	hasher := sha256.New()
	_, err = io.Copy(hasher, input)
	if err != nil {
		return "", fmt.Errorf("failed to hash file '%s': %w", path.String(), err)
	}
	hash = hasher.Sum(hash)
	hexBytes := make([]byte, len(hash)*2)
	hex.Encode(hexBytes, hash)
	return string(hexBytes[:digits]), nil
}
