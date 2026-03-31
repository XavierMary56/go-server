package storage

import "testing"

func TestDeleteProjectKeyIsSoftDelete(t *testing.T) {
	db := NewForTest(t)
	defer db.Close()

	if _, err := db.AddProjectKey("project-a", "key-a", 0); err != nil {
		t.Fatalf("add project key failed: %v", err)
	}

	if err := db.DeleteProjectKey("key-a"); err != nil {
		t.Fatalf("soft delete failed: %v", err)
	}

	keys, err := db.ListProjectKeys()
	if err != nil {
		t.Fatalf("list project keys failed: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected no active project keys after soft delete, got %d", len(keys))
	}

	enabledKey, err := db.GetEnabledProjectKey("key-a")
	if err != nil {
		t.Fatalf("get enabled project key failed: %v", err)
	}
	if enabledKey != nil {
		t.Fatal("expected soft-deleted key to be unavailable to enabled key lookup")
	}
}
