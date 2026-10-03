package main

import "testing"

func TestIsArtisanCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "artisan shell wrapper", args: []string{"./main", "artisan", "migrate"}, want: true},
		{name: "go run artisan", args: []string{"goravel", "artisan", "key:generate"}, want: true},
		{name: "serve without arguments", args: []string{"./main"}},
		{name: "server argument is not artisan", args: []string{"./main", "--version"}},
		{name: "artisan substring is not command", args: []string{"./main", "artisan:migrate"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isArtisanCommand(tt.args); got != tt.want {
				t.Fatalf("isArtisanCommand(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
