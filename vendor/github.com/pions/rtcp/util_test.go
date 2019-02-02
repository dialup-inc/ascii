package rtcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPadding(t *testing.T) {
	assert := assert.New(t)
	type testCase struct {
		input  int
		result int
	}

	cases := []testCase{
		{input: 0, result: 0},
		{input: 1, result: 3},
		{input: 2, result: 2},
		{input: 3, result: 1},
		{input: 4, result: 0},
		{input: 100, result: 0},
		{input: 500, result: 0},
	}
	for _, testCase := range cases {
		assert.Equalf(getPadding(testCase.input), testCase.result, "Test case returned wrong value for input %d", testCase.input)
	}
}
