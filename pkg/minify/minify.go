package minify

import (
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
	hashSeparator = "-"
	minifyV2Path  = "github.com/tdewolff/minify/v2/cmd/minify@%s"
	// EnvMinifyPath defines an environment variable that can be used to override the invocation path of minify.
	EnvMinifyPath = "MM_MINIFY_PATH"
)

var (
	minifyVersionPattern = regexp.MustCompile(`^(latest|v2\.\d+\.\d+)$`)
	minifierPath         string
	minifierInitOnce     sync.Once
	minifierInstallOnce  sync.Once

	//go:embed mappingFile.got
	mappingTemplateText string
	mappingTemplate     = template.Must(template.New("mappingTemplate").Parse(mappingTemplateText))
)

// ConfigFunc is a function that is able to define policy values for Minifier.
type ConfigFunc func(mini *Minifier) error

type templateFile struct {
	Package         string
	MinifiedRelPath string
	EmbedSymbol     string
	FileName        string
}

// Minifier defines consistent policies for minifying web assets.
// The type is immutable after creation with New.
type Minifier struct {
	mappingFile       mm.PathString
	assetDir          mm.PathString
	mappingFileHandle *os.File
	closeFile         sync.Once
	hashDigits        int
	minifyVersion     string
	clearBeforeWrite  bool
	packageName       string
	tasks             mm.Task
}

// HashDigits sets the number of hash digits to use for minified files.
// Digits must be between 4-32, inclusive.
func HashDigits(digits int) ConfigFunc {
	return func(mini *Minifier) error {
		if digits < 4 {
			return errors.New("hash digits must be at least 4, recommend at least 6 (default)")
		}
		if digits > 32 {
			return errors.New("hash digits cannot be greater than 32")
		}
		mini.hashDigits = digits
		return nil
	}
}

// Version sets the version of minify to use.
// Defaults to "latest".
func Version(version string) ConfigFunc {
	return func(mini *Minifier) error {
		if !minifyVersionPattern.MatchString(version) {
			return fmt.Errorf("invalid version string '%s'", version)
		}
		mini.minifyVersion = version
		return nil
	}
}

// PackageName sets the package name used in the mapping file.
// The default when no package is specified is the parent directory name.
func PackageName(packageName string) ConfigFunc {
	return func(mini *Minifier) error {
		if !token.IsIdentifier(packageName) {
			return fmt.Errorf("package name '%s' is not valid", packageName)
		}
		mini.packageName = packageName
		return nil
	}
}

// ClearBeforeWrite will make this Minifier clear the asset directory before writing any new files.
func ClearBeforeWrite() ConfigFunc {
	return func(mini *Minifier) error {
		mini.clearBeforeWrite = true
		return nil
	}
}

// New is used to create a new Minifier.
// It allows specifying policies for minifying web assets.
func New(mappingFile mm.PathString, assetDirName string, configFuncs ...ConfigFunc) (*Minifier, error) {
	if len(mappingFile) == 0 {
		return nil, errors.New("empty mapping file path")
	}
	if len(assetDirName) == 0 {
		return nil, errors.New("empty asset directory name")
	}
	assetDir := mappingFile.Dir().Join(assetDirName)
	mappingFileStr := mappingFile.String()
	if !strings.HasSuffix(mappingFileStr, ".go") {
		mappingFileStr += ".go"
		mappingFile = mm.Path(mappingFileStr)
	}
	mini := &Minifier{
		mappingFile:   mappingFile,
		assetDir:      assetDir,
		hashDigits:    6,
		minifyVersion: "latest",
	}
	var errs []error
	for _, fn := range configFuncs {
		if err := fn(mini); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(mini.packageName) == 0 {
		dir, err := mappingFile.Dir().Abs()
		if err != nil {
			return nil, err
		}
		mini.packageName = dir.Base().String()
	}
	mini.tasks = installTask(mini.minifyVersion)
	if mini.clearBeforeWrite {
		mini.tasks = mini.tasks.Then(mm.WithoutContext(func() error {
			return assetDir.RemoveAll()
		}))
	}
	mini.tasks = mini.tasks.Then(mm.WithoutContext(func() error {
		err := assetDir.Mkdir(0700)
		if err != nil {
			return fmt.Errorf("failed to create asset directory '%s': %w", assetDir, err)
		}
		mini.mappingFileHandle, err = mini.mappingFile.Create()
		if err != nil {
			return fmt.Errorf("failed to create mapping file '%s': %w", mini.mappingFile, err)
		}
		err = mappingTemplate.ExecuteTemplate(mini.mappingFileHandle, "fileHeader", &templateFile{
			Package: mini.packageName,
		})
		if err != nil {
			mini.closeFile.Do(func() {
				_ = mini.mappingFileHandle.Close()
				mini.mappingFileHandle = nil
			})
			return fmt.Errorf("failed to write mapping file header: %w", err)
		}
		return nil
	}))
	minifierInitOnce.Do(func() {
		defaultPath := mm.Path(mm.Go().GetEnv("GOBIN"), "minify").String()
		minifierPath = mm.F(fmt.Sprintf("${%s:%s}", EnvMinifyPath, defaultPath))
	})
	return mini, nil
}

func (mini *Minifier) singleFileArgs() []string {
	return []string{
		"--all",
		"--js-version", "0",
		"--html-keep-document-tags",
		"--html-keep-end-tags",
		"--html-keep-quotes",
	}
}

func (mini *Minifier) invokeMinify(source mm.PathString) mm.Task {
	return func(ctx context.Context) error {
		if mini.mappingFileHandle == nil {
			panic("unexpected state: mappingFileHandle invalid")
		}
		if !source.IsFile() {
			return fmt.Errorf("source file '%s' doesn't exist", source)
		}
		if !mini.assetDir.IsDir() {
			return fmt.Errorf("asset directory '%s' isn't valid", mini.assetDir)
		}
		fi, err := source.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat source file '%s': %w", source, err)
		}
		origSize := float64(fi.Size())
		var (
			buf strings.Builder
		)
		srcStr := source.String()
		ctx, log := mm.WithGroup(ctx, "minify "+srcStr)
		err = mini.getMinifiedContent(ctx, &buf, source)
		if err != nil {
			return err
		}
		minifiedSize := float64(buf.Len())
		if minifiedSize == 0 {
			return fmt.Errorf("no minified data written for file '%s'", source)
		}
		hash, err := hashFile(mini.hashDigits, source)
		if err != nil {
			return fmt.Errorf("failed to hash file '%s': %w", source, err)
		}
		targetFileName := hashedFileName(source, hash)
		targetFilePath := mini.assetDir.Join(targetFileName)
		embedIdentifier, err := embedSymbol(source)
		if err != nil {
			return err
		}
		relTarget, err := mini.mappingFile.Dir().Rel(targetFilePath)
		if err != nil {
			return fmt.Errorf("failed to get path to source '%s' relative to mapping file '%s': %w", source, mini.mappingFile, err)
		}
		err = targetFilePath.WriteFile([]byte(buf.String()), 0600)
		if err != nil {
			return err
		}

		err = mappingTemplate.ExecuteTemplate(mini.mappingFileHandle, "fileMapping", templateFile{
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

func (mini *Minifier) getMinifiedContent(ctx context.Context, buf *strings.Builder, source mm.PathString) error {
	err := mm.Exec(minifierPath).Stdout(buf).
		Arg(mini.singleFileArgs()...).
		TrailingArg(source.String()).
		LogGroup("minify").
		Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to invoke minify: %w", err)
	}
	return nil
}

func (mini *Minifier) Run(ctx context.Context) error {
	return mm.Print("Minifying files").Then(mini.tasks).Finally(func(terr error) error {
		var err error
		mini.closeFile.Do(func() {
			if mini.mappingFileHandle != nil {
				err = errors.Join(terr, mini.mappingFileHandle.Close())
				mini.mappingFileHandle = nil
			}
		})
		return err
	}).Run(ctx)
}

// Apply will link Minifier operations to the 'generate' step of a Modmake Build.
// Further operations may be added to the Minifier after calling Apply, but before the generate step is ran.
func (mini *Minifier) Apply(b *mm.Build) {
	b.Generate().DependsOnRunner("minify", "Minifies web asssets", mini)
}

func installTask(version string) mm.Task {
	return func(ctx context.Context) error {
		var err error
		minifierInstallOnce.Do(func() {
			ctx, _ = mm.WithGroup(ctx, "install-minify")
			err = mm.Go().Install(fmt.Sprintf(minifyV2Path, version)).Run(ctx)
		})
		return err
	}
}

func embedSymbol(source mm.PathString) (string, error) {
	base := source.Base().String()
	var buf strings.Builder
	readFirst := false
	for _, r := range base {
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

func hashedFileName(source mm.PathString, hash string) string {
	base := source.Base().String()
	ext := filepath.Ext(base)
	if len(ext) == 0 {
		return fmt.Sprintf("%s%s%s", base, hashSeparator, hash)
	}
	var buf strings.Builder
	extIdx := strings.LastIndex(base, ext)
	buf.Write([]byte(base)[:extIdx])
	buf.WriteString(hashSeparator + hash)
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
