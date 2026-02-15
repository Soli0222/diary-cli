package models

import "testing"

func strPtr(s string) *string { return &s }

func TestGetDisplayText(t *testing.T) {
	tests := []struct {
		name string
		note Note
		want string
	}{
		{
			name: "normal text",
			note: Note{Text: strPtr("hello world")},
			want: "hello world",
		},
		{
			name: "nil text",
			note: Note{},
			want: "",
		},
		{
			name: "CW with text",
			note: Note{CW: strPtr("spoiler"), Text: strPtr("secret content")},
			want: "[CW: spoiler] secret content",
		},
		{
			name: "CW without text",
			note: Note{CW: strPtr("spoiler")},
			want: "[CW: spoiler] ",
		},
		{
			name: "pure renote with text in renoted note",
			note: Note{
				RenoteID: strPtr("renote1"),
				Renote:   &Note{Text: strPtr("original post")},
			},
			want: "[RN] original post",
		},
		{
			name: "pure renote with nil text in renoted note",
			note: Note{
				RenoteID: strPtr("renote1"),
				Renote:   &Note{},
			},
			want: "",
		},
		{
			name: "quote renote (has own text)",
			note: Note{
				Text:     strPtr("my comment"),
				RenoteID: strPtr("renote1"),
				Renote:   &Note{Text: strPtr("original post")},
			},
			want: "my comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.note.GetDisplayText()
			if got != tt.want {
				t.Errorf("GetDisplayText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsOriginalNote(t *testing.T) {
	tests := []struct {
		name string
		note Note
		want bool
	}{
		{
			name: "normal note with text",
			note: Note{Text: strPtr("hello")},
			want: true,
		},
		{
			name: "note without text and without renoteID",
			note: Note{},
			want: true,
		},
		{
			name: "pure renote (no text, has renoteID)",
			note: Note{RenoteID: strPtr("renote1")},
			want: false,
		},
		{
			name: "quote renote (has text and renoteID)",
			note: Note{Text: strPtr("comment"), RenoteID: strPtr("renote1")},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.note.IsOriginalNote()
			if got != tt.want {
				t.Errorf("IsOriginalNote() = %v, want %v", got, tt.want)
			}
		})
	}
}
