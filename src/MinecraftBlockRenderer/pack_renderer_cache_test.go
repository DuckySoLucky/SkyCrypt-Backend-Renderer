package minecraftblockrenderer

import "testing"

func TestPackRendererCacheKeepsMostRecent(t *testing.T) {
	root := &MinecraftBlockRenderer{
		_packRendererCache: map[string]*MinecraftBlockRenderer{},
		_packRendererByIds: map[string]*MinecraftBlockRenderer{},
	}
	renderers := make([]*MinecraftBlockRenderer, maxPackRenderers+2)
	for i := range renderers {
		renderers[i] = &MinecraftBlockRenderer{}
		key := string(rune('a' + i))
		root._packRendererCache[key] = renderers[i]
		root._packRendererByIds["ids-"+key] = renderers[i]
		root.touchPackRendererLocked(renderers[i])
		if i == maxPackRenderers-1 {
			// Using the first one again just before the cache fills keeps it; b and c are then the oldest.
			root.touchPackRendererLocked(renderers[0])
		}
		root.evictPackRenderersLocked()
	}

	if len(root._packRendererCache) != maxPackRenderers || len(root._packRendererRecency) != maxPackRenderers {
		t.Fatalf("kept %d renderers (%d in recency), want %d", len(root._packRendererCache), len(root._packRendererRecency), maxPackRenderers)
	}
	if root._packRendererCache["a"] != renderers[0] {
		t.Fatal("the renderer used again was evicted")
	}
	for _, key := range []string{"b", "c"} {
		if _, ok := root._packRendererCache[key]; ok {
			t.Fatalf("renderer %s should have been evicted", key)
		}
		if _, ok := root._packRendererByIds["ids-"+key]; ok {
			t.Fatalf("ids entry for %s should have been dropped with it", key)
		}
	}
}
