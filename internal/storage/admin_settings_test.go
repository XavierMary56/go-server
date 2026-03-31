package storage

import "testing"

func TestAdminSettingRoundTrip(t *testing.T) {
	db := NewForTest(t)
	defer db.Close()

	setting, err := db.GetAdminSetting("admin_token_hash")
	if err != nil {
		t.Fatalf("get empty setting failed: %v", err)
	}
	if setting != nil {
		t.Fatal("expected missing setting to return nil")
	}

	if err := db.SetAdminSetting("admin_token_hash", "hash-a"); err != nil {
		t.Fatalf("set setting failed: %v", err)
	}

	setting, err = db.GetAdminSetting("admin_token_hash")
	if err != nil {
		t.Fatalf("get setting failed: %v", err)
	}
	if setting == nil || setting.Value != "hash-a" {
		t.Fatalf("expected stored value hash-a, got %#v", setting)
	}

	if err := db.SetAdminSetting("admin_token_hash", "hash-b"); err != nil {
		t.Fatalf("update setting failed: %v", err)
	}

	setting, err = db.GetAdminSetting("admin_token_hash")
	if err != nil {
		t.Fatalf("get updated setting failed: %v", err)
	}
	if setting == nil || setting.Value != "hash-b" {
		t.Fatalf("expected updated value hash-b, got %#v", setting)
	}
}
