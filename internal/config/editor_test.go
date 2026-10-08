package config

import (
	"errors"
	"reflect"
	"testing"
)

func TestResolveEditor(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		goos    string
		want    []string
		wantErr error
	}{
		{
			name: "VISUAL wins over EDITOR",
			env:  map[string]string{"VISUAL": "vim", "EDITOR": "nano"},
			goos: "linux",
			want: []string{"vim"},
		},
		{
			name: "EDITOR when VISUAL unset",
			env:  map[string]string{"EDITOR": "nano"},
			goos: "linux",
			want: []string{"nano"},
		},
		{
			name: "editor with args is split",
			env:  map[string]string{"EDITOR": "code -w"},
			goos: "darwin",
			want: []string{"code", "-w"},
		},
		{
			name: "whitespace-only env is ignored",
			env:  map[string]string{"VISUAL": "   ", "EDITOR": "nano"},
			goos: "linux",
			want: []string{"nano"},
		},
		{
			name: "darwin default",
			env:  map[string]string{},
			goos: "darwin",
			want: []string{"open", "-t"},
		},
		{
			name: "linux default",
			env:  map[string]string{},
			goos: "linux",
			want: []string{"xdg-open"},
		},
		{
			name: "windows default",
			env:  map[string]string{},
			goos: "windows",
			want: []string{"notepad"},
		},
		{
			name:    "no editor on unknown OS",
			env:     map[string]string{},
			goos:    "plan9",
			wantErr: ErrNoEditor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(k string) string { return tt.env[k] }
			got, err := ResolveEditor(getenv, tt.goos)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("argv = %v, want %v", got, tt.want)
			}
		})
	}
}
