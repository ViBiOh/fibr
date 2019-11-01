package sha

import (
	"testing"
)

func TestSha1(t *testing.T) {
	var cases = []struct {
		intention string
		input     interface{}
		want      string
	}{
		{
			"nil",
			nil,
			"3a9bcf8af38962fe340baa717773bf285f95121a",
		},
		{
			"string",
			"Hello world",
			"9b04049ba0b2dbb4ea221013dc80028992a968c7",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := Sha1(testCase.input); result != testCase.want {
				t.Errorf("Sha1(%#v) = `%s`, want `%s`", testCase.input, result, testCase.want)
			}
		})
	}
}
