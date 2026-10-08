package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetFrontmatterValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "existing value is replaced in place, everything else untouched",
			in:   "---\n# keep this comment\nname: prototype\ndescription: d\n---\n# Body\nname: not frontmatter\n",
			want: "---\n# keep this comment\nname: emil-design-prototype\ndescription: d\n---\n# Body\nname: not frontmatter\n",
		},
		{
			name: "quoted value and trailing comment are replaced",
			in:   "---\nname: \"prototype\" # upstream\n---\nBody",
			want: "---\nname: emil-design-prototype\n---\nBody",
		},
		{
			name: "nested key of the same name is not the field",
			in:   "---\nmetadata:\n  name: prototype\n---\nBody",
			want: "---\nmetadata:\n  name: prototype\nname: emil-design-prototype\n---\nBody",
		},
		{
			name: "CRLF line endings are kept",
			in:   "---\r\nname: prototype\r\ndescription: d\r\n---\r\nBody\r\n",
			want: "---\r\nname: emil-design-prototype\r\ndescription: d\r\n---\r\nBody\r\n",
		},
		{
			name: "no frontmatter gets one",
			in:   "# Just a body\n",
			want: "---\nname: emil-design-prototype\n---\n# Just a body\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "SKILL.md")
			if err := os.WriteFile(path, []byte(tt.in), 0644); err != nil {
				t.Fatal(err)
			}
			if err := SetFrontmatterValue(path, "name", "emil-design-prototype"); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got, _ := os.ReadFile(path)
			if string(got) != tt.want {
				t.Errorf("content mismatch\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
