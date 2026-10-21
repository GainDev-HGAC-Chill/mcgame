package postgres

import "testing"

func TestPetSkillSnapshotMatchesCanonicalDefaults(t *testing.T) {
	expected := []int{0, 0, 0, 0, 0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}
	if !petSkillSnapshotMatches([]byte(`{}`), expected) {
		t.Fatal("empty property did not match canonical slot defaults")
	}
	if petSkillSnapshotMatches([]byte(`{"skill1":5527}`), expected) {
		t.Fatal("stale expected snapshot matched changed skill state")
	}
	expected[0] = 5527
	if !petSkillSnapshotMatches([]byte(`{"skill1":5527}`), expected) {
		t.Fatal("exact persisted skill snapshot was rejected")
	}
}
