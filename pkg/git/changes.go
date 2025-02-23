package git

import (
	"bytes"
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake"
	"regexp"
	"strconv"
	"strings"
)

var shortStatPattern = regexp.MustCompile(`^(\d+) files? changed(, (\d+) insertions?\(\+\))?(, (\d+) deletions?\(-\))?$`)

// Changes outputs how many files were changed according to Git, with the number of insertions and deletions.
func Changes(ctx context.Context) (files, insertions, deletions int, err error) {
	var buf bytes.Buffer
	err = Exec("diff", "--shortstat").Output(&buf).Run(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	return parseShortStat(buf.String())
}

func parseShortStat(line string) (files int, insertions int, deletions int, err error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return
	}
	groups := shortStatPattern.FindStringSubmatch(line)
	files, err = strconv.Atoi(groups[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("file changed count is not a number: %w", err)
	}
	if len(groups[3]) > 0 {
		insertions, err = strconv.Atoi(groups[3])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("insertions count is not a number: %w", err)
		}
	}
	if len(groups[5]) > 0 {
		deletions, err = strconv.Atoi(groups[5])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("deletions count is not a number: %w", err)
		}
	}
	return files, insertions, deletions, nil
}

// AssertNoChanges creates a Task that returns an error if any changes in the repository are known to Git.
func AssertNoChanges() modmake.Task {
	return func(ctx context.Context) error {
		files, insertions, deletions, err := Changes(ctx)
		if err != nil {
			return err
		}
		if files > 0 {
			return fmt.Errorf("expected no changes, but %d files are changed (%d insertions and %d deletions)", files, insertions, deletions)
		}
		return nil
	}
}
