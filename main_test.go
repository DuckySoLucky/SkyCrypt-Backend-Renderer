package renderer

import "testing"

func TestNewRendererRequiresPackIDs(t *testing.T) {
	_, err := NewRenderer(Options{
		AssetsRoot:        "assets",
		ResourcePacksRoot: "resourcepacks",
	})
	if err == nil {
		t.Fatal("expected missing pack IDs to return an error")
	}
}

func TestRendererPackIDsReturnsCopy(t *testing.T) {
	source := []string{"fsr", "hplus"}
	renderer := &Renderer{packIDs: append([]string(nil), source...)}

	packIDs := renderer.PackIDs()
	packIDs[0] = "changed"

	if got := renderer.PackIDs()[0]; got != "fsr" {
		t.Fatalf("PackIDs exposed internal slice, got %q", got)
	}
}
