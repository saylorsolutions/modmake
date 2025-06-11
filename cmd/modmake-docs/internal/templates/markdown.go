package templates

import (
	"bytes"
	"context"
	"github.com/a-h/templ"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"io"
)

func Markdown(data []byte) templ.Component {
	var buf bytes.Buffer
	if err := parser().Convert(data, &buf); err != nil {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			return err
		})
	}
	data = buf.Bytes()
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		_, err := w.Write(data)
		if err != nil {
			return err
		}
		return nil
	})
}

var _ renderer.NodeRenderer = (*overrides)(nil)

type overrides struct {
	*html.Renderer
}

func (o *overrides) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	o.Renderer.RegisterFuncs(reg)
	reg.Register(ast.KindBlockquote, o.renderAside)
	reg.Register(ast.KindCodeBlock, o.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, o.renderCodeBlock)
}

func (o *overrides) writeLines(w util.BufWriter, source []byte, n ast.Node) {
	l := n.Lines().Len()
	for i := 0; i < l; i++ {
		line := n.Lines().At(i)
		o.Writer.RawWrite(w, line.Value(source))
	}
}

func (o *overrides) renderAside(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<aside>\n")
	} else {
		_, _ = w.WriteString("</aside>\n")
	}
	return ast.WalkContinue, nil
}

func (o *overrides) renderCodeBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<pre>\n")
		o.writeLines(w, source, n)
	} else {
		_, _ = w.WriteString("</pre>\n")
	}
	return ast.WalkContinue, nil
}

var parser = func() func() goldmark.Markdown {
	var _parser goldmark.Markdown
	return func() goldmark.Markdown {
		if _parser == nil {
			_renderer := renderer.NewRenderer(renderer.WithNodeRenderers(
				util.Prioritized(&overrides{
					Renderer: html.NewRenderer().(*html.Renderer),
				}, 800),
			))
			_parser = goldmark.New(
				goldmark.WithExtensions(extension.Strikethrough),
				goldmark.WithParserOptions(),
				goldmark.WithRenderer(
					_renderer,
				),
			)
		}
		return _parser
	}
}()
