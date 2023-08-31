package modmake

import (
	"context"
	"time"
)

func ExampleCommand_Silent() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Exec("go", "version").Silent().Run(ctx)
	if err != nil {
		panic(err)
	}
	// Output:
}
