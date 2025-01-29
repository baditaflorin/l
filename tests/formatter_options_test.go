// File: tests/formatter_options_test.go
package tests

import (
	"github.com/baditaflorin/l"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test Formatter WithOptions functions with 0% coverage
func TestFormatterWithOptions(t *testing.T) {
	t.Run("JSONFormatter WithOptions", func(t *testing.T) {
		formatter := l.NewJSONFormatter()
		opts := l.FormatterOptions{
			TimeFormat: "2006-01-02",
			UseColor:   true,
			Indent:     "  ",
		}

		newFormatter := formatter.WithOptions(opts)
		assert.NotNil(t, newFormatter)
	})
}
