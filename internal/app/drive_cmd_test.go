package app

import "testing"

func TestDriveBasePathDefault(t *testing.T) {
	got := driveBasePath("")
	if got != "/v1.0/me/drive" {
		t.Fatalf("driveBasePath default = %q", got)
	}
}

func TestDriveBasePathExplicit(t *testing.T) {
	got := driveBasePath("drive-123")
	if got != "/v1.0/drives/drive-123" {
		t.Fatalf("driveBasePath explicit = %q", got)
	}
}

func TestResolveDriveDownloadPathDefaultsToItemName(t *testing.T) {
	got, err := resolveDriveDownloadPath("", "report.txt", "id1")
	if err != nil {
		t.Fatalf("resolveDriveDownloadPath returned error: %v", err)
	}
	if got != "report.txt" {
		t.Fatalf("resolveDriveDownloadPath = %q, want report.txt", got)
	}
}

func TestResolveDriveDownloadPathFallsBackToID(t *testing.T) {
	got, err := resolveDriveDownloadPath("", "", "id1")
	if err != nil {
		t.Fatalf("resolveDriveDownloadPath returned error: %v", err)
	}
	if got != "id1" {
		t.Fatalf("resolveDriveDownloadPath = %q, want id1", got)
	}
}

func TestResolveDriveDownloadPathSanitizesName(t *testing.T) {
	got, err := resolveDriveDownloadPath("", "../../../etc/passwd", "id1")
	if err != nil {
		t.Fatalf("resolveDriveDownloadPath returned error: %v", err)
	}
	if got != "passwd" {
		t.Fatalf("resolveDriveDownloadPath = %q, want passwd", got)
	}
}

func TestDriveKind(t *testing.T) {
	if got := driveKind(map[string]any{"folder": map[string]any{}}); got != "folder" {
		t.Fatalf("driveKind(folder) = %q", got)
	}
	if got := driveKind(map[string]any{"file": map[string]any{}}); got != "file" {
		t.Fatalf("driveKind(file) = %q", got)
	}
	if got := driveKind(map[string]any{"id": "x"}); got != "item" {
		t.Fatalf("driveKind(item) = %q", got)
	}
}
