package git

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestChanges(t *testing.T) {
	tests := map[string]struct {
		Files      int
		Insertions int
		Deletions  int
		Line       string
	}{
		"No changes": {
			Line: "",
		},
		"Just newline": {
			Line: "\n",
		},
		"1 file, 1 insert": {
			Line:       " 1 file changed, 1 insertion(+)\n",
			Files:      1,
			Insertions: 1,
		},
		"1 file, 20 inserts": {
			Line:       " 1 file changed, 20 insertions(+)\n",
			Files:      1,
			Insertions: 20,
		},
		"1 file, 1 delete": {
			Line:      " 1 file changed, 1 deletion(-)\n",
			Files:     1,
			Deletions: 1,
		},
		"1 file, 20 deletes": {
			Line:      " 1 file changed, 20 deletions(-)\n",
			Files:     1,
			Deletions: 20,
		},
		"1 file, mixed changes": {
			Line:       " 1 file changed, 1 insertion(+), 1 deletion(-)\n",
			Files:      1,
			Insertions: 1,
			Deletions:  1,
		},
		"20 files, mixed changes": {
			Line:       " 20 files changed, 1 insertion(+), 1 deletion(-)\n",
			Files:      20,
			Insertions: 1,
			Deletions:  1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			files, insertions, deletions, err := parseShortStat(tc.Line)
			assert.NoError(t, err)
			assert.Equal(t, tc.Files, files)
			assert.Equal(t, tc.Insertions, insertions)
			assert.Equal(t, tc.Deletions, deletions)
		})
	}
}

func ExampleChanges() {
	files, inserts, deletes, err := Changes(context.Background())
	if err != nil {
		log.Fatalln("An error occurred querying changes:", err)
	}
	fmt.Printf("%d files changed, %d inserts, %d deletes\n", files, inserts, deletes)

	// Output:
	// 0 files changed, 0 inserts, 0 deletes
}
