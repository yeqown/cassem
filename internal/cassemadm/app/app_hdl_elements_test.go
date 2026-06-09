package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		compare string
		want    string
	}{
		{name: "same text", base: "feature=true", compare: "feature=true", want: "feature=true"},
		{name: "changed text", base: "feature=false", compare: "feature=true", want: "tru"},
		{name: "added line", base: "a=1\n", compare: "a=1\nb=2\n", want: "b=2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diff(tt.base, tt.compare)

			assert.Contains(t, got, tt.want)
		})
	}
}
