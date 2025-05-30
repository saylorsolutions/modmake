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
	"hash"
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

func (mini *Minifier) minAndMapFile(source mm.PathString) mm.Task {
	return func(ctx context.Context) error {
		if mini.mappingFileHandle == nil {
			panic("unexpected state: mappingFileHandle invalid")
		}
		if !mini.assetDir.IsDir() {
			panic(fmt.Sprintf("asset directory '%s' isn't valid", mini.assetDir))
		}
		var buf bytes.Buffer
		origSize, minifiedSize, err := mini.minifyFileToBuffer(ctx, &buf, source)
		if err != nil {
			return err
		}
		target, err := mini.getAssetTargetPath(source)
		if err != nil {
			return err
		}
		err = target.WriteFile(buf.Bytes(), 0600)
		if err != nil {
			return fmt.Errorf("failed to write data to target: %w", err)
		}
		err = mini.writeTargetMapping(source, target)
		if err != nil {
			return err
		}

		reduction := (origSize - minifiedSize) / origSize * 100.0
		mm.GetLogger(ctx).Info("File '%s' size reduced by %0.2f%% and written to '%s'\n", source, reduction, target)
		return nil
	}
}

func (mini *Minifier) minAndMapBundle(bundleName string, sources ...mm.PathString) mm.Task {
	if len(sources) == 0 {
		return mm.Error("no source files given to bundle")
	}
	return func(ctx context.Context) error {
		if len(bundleName) == 0 {
			panic("missing bundle name")
		}
		ext := filepath.Ext(bundleName)
		if len(ext) == 0 {
			panic("missing bundle file extension")
		}
		if mini.mappingFileHandle == nil {
			panic("unexpected state: mappingFileHandle invalid")
		}
		if !mini.assetDir.IsDir() {
			panic(fmt.Sprintf("asset directory '%s' isn't valid", mini.assetDir))
		}
		var (
			buf                              bytes.Buffer
			totalOrigSize, totalMinifiedSize float64
			hasher                           = newFileHasher()
		)
		for _, source := range sources {
			origSize, minifiedSize, err := mini.minifyFileToBuffer(ctx, &buf, source)
			if err != nil {
				return err
			}
			hasher.hashFile(source)

			totalOrigSize += origSize
			totalMinifiedSize += minifiedSize
			reduction := (origSize - minifiedSize) / origSize * 100.0
			mm.GetLogger(ctx).Info("Bundle file '%s' size reduced by %0.2f%%\n", source, reduction)
		}
		hashDigits, err := hasher.getHashDigits(mini.hashDigits)
		if err != nil {
			return err
		}
		target := mini.assetDir.Join(hashedFileName(mm.Path(bundleName), hashDigits))
		err = target.WriteFile(buf.Bytes(), 0600)
		if err != nil {
			return fmt.Errorf("failed to write bundle data to target '%s': %w", target, err)
		}
		err = mini.writeBundleMapping(bundleName, target)
		if err != nil {
			return err
		}
		reduction := (totalOrigSize - totalMinifiedSize) / totalOrigSize * 100.0
		mm.GetLogger(ctx).Info("Total size reduced by %0.2f%% and written to '%s'\n", reduction, target)
		return nil
	}
}

func (mini *Minifier) writeTargetMapping(source mm.PathString, target mm.PathString) error {
	embedIdentifier, err := embedSymbolFromSource(source)
	if err != nil {
		return err
	}
	relTarget, err := mini.mappingFile.Dir().Rel(target)
	if err != nil {
		return fmt.Errorf("failed to get path to source '%s' relative to mapping file '%s': %w", source, mini.mappingFile, err)
	}

	err = mappingTemplate.ExecuteTemplate(mini.mappingFileHandle, "fileMapping", templateFile{
		MinifiedRelPath: relTarget.ToSlash(),
		EmbedSymbol:     embedIdentifier,
		FileName:        target.Base().String(),
	})
	if err != nil {
		return fmt.Errorf("failed to write embed mapping to mapping file: %w", err)
	}
	return nil
}

func (mini *Minifier) writeBundleMapping(bundleName string, target mm.PathString) error {
	embedIdentifier, err := embedSymbol(bundleName)
	if err != nil {
		return err
	}
	relTarget, err := mini.mappingFile.Dir().Rel(target)
	if err != nil {
		return fmt.Errorf("failed to get path to bundle target '%s' relative to mapping file '%s': %w", target, mini.mappingFile, err)
	}

	err = mappingTemplate.ExecuteTemplate(mini.mappingFileHandle, "fileMapping", templateFile{
		MinifiedRelPath: relTarget.ToSlash(),
		EmbedSymbol:     embedIdentifier,
		FileName:        target.Base().String(),
	})
	if err != nil {
		return fmt.Errorf("failed to write embed mapping to mapping file: %w", err)
	}
	return nil
}

func (mini *Minifier) getAssetTargetPath(source mm.PathString) (mm.PathString, error) {
	hashDigits, err := newFileHasher().hashFile(source).getHashDigits(mini.hashDigits)
	if err != nil {
		return "", fmt.Errorf("failed to hash file '%s': %w", source, err)
	}
	return mini.assetDir.Join(hashedFileName(source, hashDigits)), nil
}

func (mini *Minifier) minifyFileToBuffer(ctx context.Context, buf *bytes.Buffer, source mm.PathString) (origSize float64, minifiedSize float64, err error) {
	return mini.minifyFile(ctx, buf, source)
}

type countingWriter struct {
	wrapped io.Writer
	count   int64
}

func (w *countingWriter) Write(data []byte) (int, error) {
	count, err := w.wrapped.Write(data)
	w.count += int64(count)
	if err != nil {
		return count, err
	}
	return count, nil
}

func (mini *Minifier) minifyFile(ctx context.Context, out io.Writer, source mm.PathString) (origSize float64, minifiedSize float64, err error) {
	counter := &countingWriter{wrapped: out}
	fi, err := source.Stat()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to stat source file '%s': %w", source, err)
	}
	if fi.IsDir() {
		return 0, 0, fmt.Errorf("source '%s' is a directory", source)
	}
	origSize = float64(fi.Size())
	err = mini.getMinifiedContent(ctx, counter, source)
	if err != nil {
		return 0, 0, err
	}
	minifiedSize = float64(counter.count)
	if minifiedSize == 0 {
		return 0, 0, fmt.Errorf("no minified data written for file '%s'", source)
	}
	return origSize, minifiedSize, nil
}

func (mini *Minifier) getMinifiedContent(ctx context.Context, out io.Writer, source mm.PathString) error {
	err := mm.Exec(minifierPath).Stdout(out).
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

func embedSymbolFromSource(source mm.PathString) (string, error) {
	base := source.Base().String()
	return embedSymbol(base)
}

func embedSymbol(fileName string) (string, error) {
	var buf strings.Builder
	readFirst := false
	for _, r := range fileName {
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
		return "", fmt.Errorf("unable to derive embed symbol from file name '%s'", fileName)
	}
	return id, nil
}

type fileHasher struct {
	hash.Hash
	err error
}

func (f *fileHasher) hashFile(source mm.PathString) *fileHasher {
	if f.err != nil {
		return f
	}
	src, err := source.Open()
	if err != nil {
		f.err = fmt.Errorf("failed to open '%s' for reading: %w", source, err)
		return f
	}
	defer func() {
		_ = src.Close()
	}()
	if err := f.hashReader(src).err; err != nil {
		f.err = fmt.Errorf("failed to read from '%s': %w", source, err)
		return f
	}
	return f
}

func (f *fileHasher) hashReader(r io.Reader) *fileHasher {
	_, err := io.Copy(f.Hash, r)
	if err != nil {
		f.err = err
		return f
	}
	return f
}

func (f *fileHasher) getHashDigits(digits int) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	sum := f.Sum(nil)
	return string([]rune(hex.EncodeToString(sum))[:digits]), nil
}

func newFileHasher() *fileHasher {
	return &fileHasher{
		Hash: sha256.New(),
	}
}

func hashedFileName(source mm.PathString, hashDigits string) string {
	base := source.Base().String()
	ext := filepath.Ext(base)
	if len(ext) == 0 {
		return fmt.Sprintf("%s%s%s", base, hashSeparator, hashDigits)
	}
	var buf strings.Builder
	extIdx := strings.LastIndex(base, ext)
	buf.Write([]byte(base)[:extIdx])
	buf.WriteString(hashSeparator + hashDigits)
	buf.WriteString(ext)
	return buf.String()
}
