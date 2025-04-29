package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type timestampTest struct {
	input    string
	expected string
}

var timestampTests = []timestampTest{
	{
		input:    "2025-04-24-14",
		expected: "2025-04-24 14:00:00 +0000 UTC",
	},
	{
		input:    "2025-04-24",
		expected: "2025-04-24 00:00:00 +0000 UTC",
	},
	{
		input:    "2025-04",
		expected: "2025-04-01 00:00:00 +0000 UTC",
	},
	{
		input:    "2025",
		expected: "2025-01-01 00:00:00 +0000 UTC",
	},
}

func TestParseTimestamp(t *testing.T) {
	for _, test := range timestampTests {
		output, err := parseTimestamp(test.input)
		require.NoError(t, err)
		assert.Equal(t, test.expected, output.String())
	}
}
