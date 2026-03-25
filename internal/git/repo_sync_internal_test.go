package git

import "testing"

func TestStateHasRemoteUpdates(t *testing.T) {
	tests := []struct {
		name  string
		state RemoteSyncState
		want  bool
	}{
		{name: "up to date", state: RemoteSyncUpToDate, want: false},
		{name: "remote ahead", state: RemoteSyncRemoteAhead, want: true},
		{name: "remote significantly ahead", state: RemoteSyncRemoteSignificantlyAhead, want: true},
		{name: "local ahead", state: RemoteSyncLocalAhead, want: false},
		{name: "diverged", state: RemoteSyncDiverged, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stateHasRemoteUpdates(tt.state); got != tt.want {
				t.Fatalf("stateHasRemoteUpdates(%q) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}
