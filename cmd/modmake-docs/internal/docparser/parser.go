package docparser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Parser struct {
	goExec string
}

func New() *Parser {
	goExec, err := exec.LookPath("go")
	if err != nil || len(goExec) == 0 {
		goExec = "go"
	}
	if goRoot, ok := os.LookupEnv("GOROOT"); ok {
		goExec = filepath.Join(goRoot, "bin", "go")
	}
	return &Parser{goExec: goExec}
}

func (p *Parser) runGo(args ...string) *exec.Cmd {
	return exec.Command(p.goExec, args...) //nolint:gosec
}

type parserState int

const (
	rPackage parserState = iota + 1
	rComment
	rVariables
	rFunctions
	rMethods
	rTypes
	rConsts
)

// ParsePackageDir parses a package directory and returns [PackageDocs] representing the exposed symbols found.
func (p *Parser) ParsePackageDir(pkgDir string) (*PackageDocs, error) {
	output, err := p.runGo("doc", "-all", pkgDir).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get package docs for directory '%s': %w", pkgDir, err)
	}
	docs, err := parseDocOutput(output)
	if err != nil {
		return nil, err
	}
	if len(docs.PackageName) == 0 || len(docs.ImportName) == 0 {
		return nil, errors.New("empty package name")
	}
	return docs, nil
}

var (
	packagePattern    = regexp.MustCompile(`^package (\S+) // import "([^"]+)"$`)
	commentPattern    = regexp.MustCompile(`^( {4})(.+)$`)
	blockVarStart     = regexp.MustCompile(`^var \($`)
	blockVarPattern   = regexp.MustCompile(`^\s+(\S+)[^/]+(// (.+))?$`)
	blockVarEnd       = regexp.MustCompile(`^\)$`)
	lineVarPattern    = regexp.MustCompile(`^var (\S+)[^/]+(// (.+))?$`)
	methodPattern     = regexp.MustCompile(`^func \([A-Za-z0-9_]+ \*?([^\s)]+)\) ([^(\[]+).+$`)
	funcPattern       = regexp.MustCompile(`^func ([^(\[]+).+$`)
	typeStart         = regexp.MustCompile(`^type (\S+).+$`)
	typeFieldPattern  = regexp.MustCompile(`^\t(.+)$`)
	typeEnd           = regexp.MustCompile(`^}$`)
	blockConstStart   = regexp.MustCompile(`^const \($`)
	blockConstPattern = regexp.MustCompile(`^\t(\S+)[^/]+(// (.+))?$`)
	blockConstEnd     = regexp.MustCompile(`^\)$`)
	lineConstPattern  = regexp.MustCompile(`^const (\S+)[^/]+(// (.+))?$`)
	boundaryPatterns  = []*regexp.Regexp{
		packagePattern,
		blockVarStart,
		lineVarPattern,
		funcPattern,
		methodPattern,
		typeStart,
		blockConstStart,
		lineConstPattern,
	}
	boundaryStates = []parserState{
		rPackage,
		rVariables,
		rVariables,
		rFunctions,
		rMethods,
		rTypes,
		rConsts,
		rConsts,
	}
)

func isBoundary(line []byte) (parserState, bool) {
	for i, pat := range boundaryPatterns {
		if pat.Match(line) {
			return boundaryStates[i], true
		}
	}
	return 0, false
}

func parseDocOutput(output []byte) (*PackageDocs, error) {
	var (
		commentBuf    bytes.Buffer
		commentTarget *string
		docs          = PackageDocs{
			Functions: map[string]*Function{},
			Types:     map[string]*Type{},
		}
		state    = rPackage
		line     []byte
		lastType *Type
	)
	flushComment := func() {
		comment := strings.TrimSpace(commentBuf.String())
		commentBuf.Reset()
		if len(comment) > 0 && commentTarget != nil {
			*commentTarget = comment
		}
		commentTarget = nil
	}
	setTarget := func(target *string) {
		flushComment()
		commentTarget = target
	}
	consumeLine := func() {
		line = nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for {
		if len(line) == 0 {
			if !scanner.Scan() {
				break
			}
			line = scanner.Bytes()
		}

		switch string(line) {
		case "CONSTANTS":
			state = rConsts
			consumeLine()
			continue
		case "VARIABLES":
			state = rVariables
			consumeLine()
			continue
		case "FUNCTIONS":
			state = rFunctions
			consumeLine()
			continue
		case "TYPES":
			state = rTypes
			consumeLine()
			continue
		}

		switch state {
		case rPackage:
			if len(line) == 0 {
				continue
			}
			if !packagePattern.Match(line) {
				return nil, fmt.Errorf("expected package pattern, found '%s'", string(line))
			}
			groups := packagePattern.FindSubmatch(line)
			docs.PackageName = string(groups[1])
			docs.ImportName = string(groups[2])
			setTarget(&docs.Docs)
			state = rComment
			consumeLine()
		case rComment:
			if newState, ok := isBoundary(line); ok {
				state = newState
				flushComment()
				continue
			}
			if commentPattern.Match(line) {
				groups := commentPattern.FindSubmatch(line)
				commentBuf.Write(groups[2])
				commentBuf.WriteString("\n")
				consumeLine()
				continue
			}
			commentBuf.Write(line)
			commentBuf.WriteString("\n")
			consumeLine()
		case rVariables:
			if blockVarEnd.Match(line) || blockVarStart.Match(line) {
				flushComment()
				consumeLine()
				continue
			}
			if blockVarPattern.Match(line) {
				groups := blockVarPattern.FindSubmatch(line)
				_var := &Variable{
					VarName:     string(groups[1]),
					Declaration: string(groups[0]),
					Docs:        string(groups[3]),
				}
				docs.Variables = append(docs.Variables, _var)
				consumeLine()
				continue
			}
			if lineVarPattern.Match(line) {
				groups := lineVarPattern.FindSubmatch(line)
				_var := &Variable{
					VarName:     string(groups[1]),
					Declaration: string(groups[0]),
					Docs:        string(groups[3]),
				}
				docs.Variables = append(docs.Variables, _var)
				setTarget(&_var.Docs)
				consumeLine()
			}
			state = rComment
		case rFunctions:
			if funcPattern.Match(line) {
				groups := funcPattern.FindSubmatch(line)
				_func := &Function{
					FunctionName: string(groups[1]),
					Signature:    string(line),
				}
				docs.Functions[_func.FunctionName] = _func
				setTarget(&_func.Docs)
				consumeLine()
			}
			state = rComment
		case rTypes:
			if typeStart.Match(line) {
				flushComment()
				groups := typeStart.FindSubmatch(line)
				typ := &Type{
					TypeName:    string(groups[1]),
					Declaration: string(line),
					Methods:     map[string]*Method{},
				}
				lastType = typ
				docs.Types[typ.TypeName] = typ
				consumeLine()
				setTarget(&typ.Docs)
				continue
			}
			if typeFieldPattern.Match(line) && lastType != nil {
				lastType.Declaration += "\n" + string(line)
				consumeLine()
				continue
			}
			if typeEnd.Match(line) && lastType != nil {
				lastType.Declaration += "\n}"
				consumeLine()
			}
			state = rComment
		case rMethods:
			if methodPattern.Match(line) {
				flushComment()
				groups := methodPattern.FindSubmatch(line)
				typName, methodName := string(groups[1]), string(groups[2])
				method := &Method{
					MethodName: methodName,
					TypeName:   typName,
					Signature:  string(line),
				}
				setTarget(&method.Docs)
				docs.Types[typName].Methods[methodName] = method
				consumeLine()
			}
			state = rComment
		case rConsts:
			if blockConstStart.Match(line) {
				flushComment()
				if lastType != nil {
					lastType.ConstVals = "\nconst ("
				}
				consumeLine()
				continue
			}
			if blockConstPattern.Match(line) {
				if lastType != nil {
					lastType.ConstVals += "\n" + string(line)
					consumeLine()
					continue
				}
				groups := blockConstPattern.FindSubmatch(line)
				name, comment := string(groups[1]), string(groups[3])
				docs.Constants = append(docs.Constants, &Constant{
					ConstantName: name,
					Declaration:  strings.TrimSpace(string(line)),
					Docs:         comment,
				})
				consumeLine()
				continue
			}
			if blockConstEnd.Match(line) && lastType != nil {
				consumeLine()
				lastType.ConstVals = strings.TrimSpace(lastType.ConstVals + "\n}")
			}
			if lineConstPattern.Match(line) {
				groups := lineConstPattern.FindSubmatch(line)
				_const := &Constant{
					ConstantName: string(groups[1]),
					Declaration:  string(groups[0]),
					Docs:         string(groups[3]),
				}
				if lastType == nil {
					docs.Constants = append(docs.Constants, _const)
				} else {
					lastType.ConstVals += "\n" + string(line)
				}
				consumeLine()
			}
			state = rComment
		}
	}
	flushComment()
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return &docs, nil
}
