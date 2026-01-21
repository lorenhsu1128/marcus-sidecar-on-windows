package filebrowser

import "testing"

func TestTabs_OpenTabReplaceCreatesTab(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	cmd := p.openTab("main.go", TabOpenReplace)
	if cmd == nil {
		t.Error("expected LoadPreview command for new tab")
	}
	if len(p.tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(p.tabs))
	}
	if p.activeTab != 0 {
		t.Errorf("expected activeTab 0, got %d", p.activeTab)
	}
	if p.previewFile != "main.go" {
		t.Errorf("expected previewFile main.go, got %s", p.previewFile)
	}
}

func TestTabs_OpenTabNewSavesScroll(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.tabs = []FileTab{{Path: "main.go"}}
	p.activeTab = 0
	p.previewFile = "main.go"
	p.previewScroll = 7

	_ = p.openTab("src/app.go", TabOpenNew)

	if len(p.tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(p.tabs))
	}
	if p.activeTab != 1 {
		t.Errorf("expected activeTab 1, got %d", p.activeTab)
	}
	if p.tabs[0].Scroll != 7 {
		t.Errorf("expected saved scroll 7, got %d", p.tabs[0].Scroll)
	}
}

func TestTabs_CloseTabSelectsNext(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.tabs = []FileTab{
		{Path: "main.go", Loaded: true, Result: PreviewResult{Lines: []string{"a"}}},
		{Path: "src/app.go", Loaded: true, Result: PreviewResult{Lines: []string{"b"}}},
	}
	p.activeTab = 0
	p.previewFile = "main.go"

	_ = p.closeTab(0)

	if len(p.tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(p.tabs))
	}
	if p.previewFile != "src/app.go" {
		t.Errorf("expected previewFile src/app.go, got %s", p.previewFile)
	}
}

func TestTabs_CloseLastTabClearsPreview(t *testing.T) {
	tmpDir := t.TempDir()
	p := createTestPlugin(t, tmpDir)

	p.tabs = []FileTab{{Path: "main.go"}}
	p.activeTab = 0
	p.previewFile = "main.go"

	_ = p.closeTab(0)

	if len(p.tabs) != 0 {
		t.Fatalf("expected 0 tabs, got %d", len(p.tabs))
	}
	if p.previewFile != "" {
		t.Errorf("expected previewFile cleared, got %s", p.previewFile)
	}
}
