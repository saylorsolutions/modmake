package modmake

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
)

// EnvMap is a specialized map for holding environment variables that are used to interpolate strings.
type EnvMap map[string]string

func (m EnvMap) merge(other EnvMap) EnvMap {
	if other == nil {
		return m
	}
	for k, v := range other {
		if len(k) == 0 {
			continue
		}
		m[strings.ToUpper(k)] = v
	}
	return m
}

// Environment returns the currently set environment values as an EnvMap.
func Environment() EnvMap {
	m := EnvMap{}
	for _, entry := range os.Environ() {
		kv := strings.SplitN(entry, "=", 2)
		if len(kv) != 2 {
			m[strings.ToUpper(kv[0])] = ""
		}
		m[strings.ToUpper(kv[0])] = kv[1]
	}
	return m
}

// F will format a string, replacing variables with their value as found in the environment data.
// Additional values may be added as the second parameter, which will override values in the original environment.
//
// Variable placeholders may be specified like ${ENV_VAR_NAME}. The example below will replace ${BUILD_NUM} with a value from the environment, or an empty string.
// If a variable either doesn't exist in the environment, or has an empty value, then an empty string will replace the variable placeholder.
//
//	str := F("My string that references build ${BUILD_NUM}")
//
// Note that the "${" prefix and "}" suffix are required, but the variable name may be space padded for readability if desired.
// Also, variable names are case insensitive.
//
// A default value for interpolation can be specified with a colon (":") separator after the key.
// In the example below, if BUILD_NUM was not defined, then the string "0" would be used instead.
//
//	str := F("My string that references build ${BUILD_NUM:0}")
//
// Note that whitespace characters in default values will always be trimmed.
// The string "${" can still be expressed in F strings, but it must be formatted to use a default value, which means that interpolation cannot be recursive.
// The string output from the function call below will be "My string has a variable reference ${BUILD_NUM}".
//
//	str := F("My string has a variable reference ${:$}{BUILD_NUM}")
func F(fmt string, data ...EnvMap) string {
	return string(FReader(strings.NewReader(fmt), data...))
}

// FReader will do the same thing as F, but operates on an io.Reader expressing a stream of UTF-8 encoded bytes instead.
func FReader(in io.Reader, data ...EnvMap) []byte {
	var rr io.RuneReader
	if _rr, ok := in.(io.RuneReader); ok {
		rr = _rr
	} else {
		rr = bufio.NewReader(in)
	}
	m := Environment()
	if len(data) > 0 {
		m.merge(data[0])
	}
	return parseString(rr, m)
}

func parseString(in io.RuneReader, data EnvMap) []byte {
	const (
		DOLLAR rune = '$'
		BRACE  rune = '{'
	)
	var (
		outBuf bytes.Buffer
	)
	for {
		r, _, err := in.ReadRune()
		if err != nil {
			return outBuf.Bytes()
		}
		switch r {
		case DOLLAR:
			maybeBrace, _, err := in.ReadRune()
			if err != nil {
				outBuf.WriteRune(r)
				return outBuf.Bytes()
			}
			if maybeBrace == BRACE {
				outBuf.Write(replaceIdentifier(in, data))
			} else {
				outBuf.WriteRune(r)
				outBuf.WriteRune(maybeBrace)
			}
		default:
			outBuf.WriteRune(r)
		}
	}
}

func replaceIdentifier(in io.RuneReader, data EnvMap) []byte {
	const (
		END_BRACE = '}'
	)
	var (
		varBuf       strings.Builder
		defaultValue string
	)
	for {
		r, _, err := in.ReadRune()
		if err != nil {
			return nil
		}
		if r == END_BRACE {
			baseKey := strings.TrimSpace(varBuf.String())
			if before, after, found := strings.Cut(baseKey, ":"); found {
				baseKey, defaultValue = strings.TrimSpace(before), strings.TrimSpace(after)
			}
			baseKey = strings.ToUpper(baseKey)
			if val, ok := data[baseKey]; ok {
				return []byte(val)
			}
			return []byte(defaultValue)
		}
		varBuf.WriteRune(r)
	}
}
