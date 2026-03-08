package episodestore

import "testing"

func TestBaseDirFromDSN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dsn  string
		want string
	}{
		{name: "empty uses data dir", dsn: "", want: "./data"},
		{name: "basename uses data dir", dsn: "universe.db", want: "./data"},
		{name: "relative path keeps parent", dsn: "./state/universe.db", want: "state"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BaseDirFromDSN(tt.dsn); got != tt.want {
				t.Fatalf("BaseDirFromDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}
