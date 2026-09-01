package fuzzer

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestStreamProcess(t *testing.T) {
	tests := []struct {
		name  string
		input io.Reader
		want  string
	}{
		{
			name:  "single line",
			input: strings.NewReader("hello"),
			want:  "hello\n",
		},
		{
			name:  "multiple lines",
			input: strings.NewReader("line1\nline2\nline3"),
			want:  "line1\nline2\nline3\n",
		},
		{
			name:  "empty input",
			input: strings.NewReader(""),
			want:  "",
		},
		{
			name:  "trailing newline preserved",
			input: strings.NewReader("abc\n"),
			want:  "abc\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(make([]byte, 0, 4096))
			done := make(chan struct{})
			go streamProcess(tt.input, buf, done)
			<-done
			if got := buf.String(); got != tt.want {
				t.Errorf("streamProcess() = %q, want %q", got, tt.want)
			}
		})
	}
}
