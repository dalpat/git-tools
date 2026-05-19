package gitdata

import (
	"reflect"
	"testing"
)

func TestParseTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Tag
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "single tag",
			input:    "v1.0.0\x00abc1234",
			expected: []Tag{{Name: "v1.0.0", CommitHash: "abc1234"}},
		},
		{
			name:  "multiple tags",
			input: "v1.0.0\x00abc1234\nv1.1.0\x00def5678\nv2.0.0\x00ghi9012",
			expected: []Tag{
				{Name: "v1.0.0", CommitHash: "abc1234"},
				{Name: "v1.1.0", CommitHash: "def5678"},
				{Name: "v2.0.0", CommitHash: "ghi9012"},
			},
		},
		{
			name:     "tag with annotations",
			input:    "release/v2.0.0\x00a1b2c3d4",
			expected: []Tag{{Name: "release/v2.0.0", CommitHash: "a1b2c3d4"}},
		},
		{
			name:     "tag with newline in input",
			input:    "v1.0.0\x00abc1234\n",
			expected: []Tag{{Name: "v1.0.0", CommitHash: "abc1234"}},
		},
		{
			name:     "tag with spaces in name",
			input:    "version 1.0\x00abc1234",
			expected: []Tag{{Name: "version 1.0", CommitHash: "abc1234"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTags(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseTags(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetTags(t *testing.T) {
	tests := []struct {
		name     string
		responses map[string]string
		expected []Tag
		expectErr bool
	}{
		{
			name: "successful fetch",
			responses: map[string]string{
				"for-each-ref --format=%(refname:short)%x00%(objectname:short) refs/tags/": "v1.0.0\x00abc1234\nv1.1.0\x00def5678",
			},
			expected: []Tag{
				{Name: "v1.0.0", CommitHash: "abc1234"},
				{Name: "v1.1.0", CommitHash: "def5678"},
			},
			expectErr: false,
		},
		{
			name: "no tags",
			responses: map[string]string{
				"for-each-ref --format=%(refname:short)%x00%(objectname:short) refs/tags/": "",
			},
			expected:  nil,
			expectErr: false,
		},
		{
			name: "single tag",
			responses: map[string]string{
				"for-each-ref --format=%(refname:short)%x00%(objectname:short) refs/tags/": "v2.0.0\x00xyz7890",
			},
			expected: []Tag{
				{Name: "v2.0.0", CommitHash: "xyz7890"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockRunner{
				responses: tt.responses,
			}
			gd := New(mock)
			
			result, err := gd.GetTags()
			
			if tt.expectErr && err == nil {
				t.Errorf("GetTags() expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("GetTags() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetTags() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagStruct(t *testing.T) {
	tag := Tag{
		Name:       "v1.0.0",
		CommitHash: "abc123def456",
	}
	
	if tag.Name != "v1.0.0" {
		t.Errorf("Tag.Name = %q, want %q", tag.Name, "v1.0.0")
	}
	
	if tag.CommitHash != "abc123def456" {
		t.Errorf("Tag.CommitHash = %q, want %q", tag.CommitHash, "abc123def456")
	}
}
