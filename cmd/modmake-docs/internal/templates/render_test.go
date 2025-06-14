package templates

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMain_Render(t *testing.T) {
	ctx, buf, params := testParams()
	require.NoError(t, Main(params).Render(ctx, &buf))
	t.Log(buf.String())
}

func testParams() (context.Context, strings.Builder, Params) { //nolint:unparam // False positive
	p := Params{
		LatestGoVersion:          "1.22",
		LatestSupportedGoVersion: "1.20",
		ModmakeVersion:           "0.4.0",
	}
	p.Content = Content{
		Sections: []*Section{
			IntroSection(p),
			BuildModelSection(p),
			ModmakeCLISection(p),
			UtilitiesSection(),
		},
	}
	var buf strings.Builder
	return context.Background(), buf, p
}
