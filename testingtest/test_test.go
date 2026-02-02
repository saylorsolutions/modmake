package testingtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEasy(t *testing.T) {
	assert.Equal(t, 1, 1) //nolint:testifylint // Intentionally trivial test.
}

func BenchmarkEasy(b *testing.B) {
	num := 1
	for i := 0; i < b.N; i++ {
		num = 3*num + 1
		if num%2 == 0 {
			num /= 2
		}
	}
}
