# SkyCrypt-Backend-Renderer

[![Go Reference](https://pkg.go.dev/badge/github.com/DuckySoLucky/SkyCrypt-Backend-Renderer.svg)](https://pkg.go.dev/github.com/DuckySoLucky/SkyCrypt-Backend-Renderer)

Renders Minecraft and Hypixel SkyBlock items to WebP images, the way the game shows them in an inventory, with the
vanilla textures or a stack of resource packs (Hypixel Plus, FurSky Reborn, ...). It renders:

- items by SkyBlock ID (`HYPERION`, `ASPECT_OF_THE_END`) or from their decoded NBT, so custom textures that depend on
  the item's data apply;
- 3D block items, flat items, player heads from their skin texture, dyed leather, potions;
- animated textures, as animated WebP.

Rendered images are written to a cache directory and reused, so each item is drawn once.
[SkyCrypt-Backend](https://github.com/SkyCryptWebsite/SkyCrypt-Backend) uses it for every item icon on the site.

## Installation

```bash
go get github.com/DuckySoLucky/SkyCrypt-Backend-Renderer
```

## What it needs on disk

```text
resourcepacks/                     ResourcePacksRoot
├── Vanilla/
│   └── assets/minecraft/          AssetsRoot: the client's extracted assets (models, textures, items, ...)
├── Hypixel_Plus/
│   ├── meta.json                  {"id": "HYPIXEL_PLUS", "name": "Hypixel Plus", "priority": 900, ...}
│   └── Hypixel+ ....cats.zip      the pack itself: a folder, a .zip or a .cats.zip
└── FurSky_Reborn/
    ├── meta.json                  {"id": "FSR", ...}
    └── FurSky Reborn.cats.zip
cache/                             CacheDir: rendered images go here
```

- **`AssetsRoot`** is the `assets/minecraft` folder of a Minecraft client jar. This repository ships one under
  `packs/assets/minecraft`.
- **`ResourcePacksRoot`** holds one folder per pack. Each pack is registered under the `id` in its `meta.json`; that
  id is what you pass in `PackIDs`.
- **`CacheDir`** is created if missing. Renders are stored under `rendered/` (plus `player_skins/` and `derived/`). A
  cache written by another version of the renderer is wiped automatically.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	renderer "github.com/DuckySoLucky/SkyCrypt-Backend-Renderer"
)

func main() {
	r, err := renderer.NewRendererWithOptions(renderer.Options{
		AssetsRoot:        "resourcepacks/Vanilla/assets/minecraft",
		ResourcePacksRoot: "resourcepacks",
		PackIDs:           []string{"FSR", "HYPIXEL_PLUS"}, // later packs override earlier ones
		CacheDir:          "cache",
		Size:              128, // pixels; 0 means renderer.DefaultSize (128)
	})
	if err != nil {
		log.Fatal(err)
	}

	item, err := r.RenderSkyBlockItemID("HYPERION")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(item.Path)          // cache/rendered/skyblock=HYPERION__pack=FSR__...__hash=....webp
	fmt.Println(item.TexturePackID) // which pack the texture came from, or "vanilla"
}
```

The renderer is safe for concurrent use. Build it once at startup: loading the assets and packs takes a second or
two and tens of MB. Concurrent requests for the same image render it once.

## Rendering items from NBT

Items in a player's inventory can look different from their base item: a pet skin, a dye, a custom name or a pack
texture keyed on `ExtraAttributes`. Render those from the item's NBT, as a map with the decoded fields:

```go
// A dyed leather chestplate. Player heads work the same way, with
// tag.SkullOwner.Properties.textures[0].Value set to the skin texture.
item := map[string]any{
	"id": "minecraft:leather_chestplate",
	"tag": map[string]any{
		"ExtraAttributes": map[string]any{"id": "FARM_SUIT_CHESTPLATE"},
		"display":         map[string]any{"color": 0xFEE15C},
	},
}

rendered, err := r.RenderItemNBT(item)
if err != nil {
	log.Fatal(err)
}
fmt.Println(rendered.Path)
```

The input is read loosely, so the decoded NBT of an inventory (for example a
[SkyCrypt-Types](https://github.com/DuckySoLucky/SkyCrypt-Types) `Item` turned into a map) works as is. The fields it
looks at:

| Field | Meaning |
|---|---|
| `id` | Minecraft item ID (`minecraft:diamond_sword`) or a legacy numeric ID |
| `Damage` / `damage` | Legacy data value (wool colour, potion type, ...) |
| `tag.ExtraAttributes.id` | SkyBlock item ID, used by resource packs to pick a texture |
| `tag.ExtraAttributes.*` | Anything else a pack's texture rules check (`modifier`, `model`, enchantments, ...) |
| `tag.display.Name`, `tag.display.color` | Custom name; leather dye colour |
| `tag.SkullOwner.Properties.textures[0].Value` | Player head skin |
| `ItemModel` | Explicit 1.21 item model, when the item has one |

## Choosing packs per render

`PackIDs` in `Options` is the default stack. To render with another stack (for example the packs a visitor enabled),
pass it per call. Order matters: **later IDs take priority**, the way Minecraft applies the pack list bottom to top.
An empty list renders vanilla.

```go
userPacks := []string{"HYPIXEL_PLUS", "FSR"} // FSR wins where both have a texture

rendered, err := r.RenderSkyBlockItemIDWithPackIDs("ASPECT_OF_THE_END", userPacks)
rendered, err = r.RenderItemNBTWithPackIDs(item, userPacks)
vanilla, err := r.RenderSkyBlockItemIDWithPackIDs("ASPECT_OF_THE_END", []string{})
```

Each distinct stack gets its own internal renderer with that stack's models loaded. Up to 8 are kept; past that the
least recently used one is dropped and rebuilt if it is asked for again. Every pack order counts as a separate stack,
so normalize the order (for example by each pack's `priority`) if you let users pick packs.

## Pre-rendering

To render a whole catalogue ahead of time (every SkyBlock item, after a pack update), render in parallel into the
cache and read the summary:

```go
ids := []string{"HYPERION", "TERMINATOR", "NECRON_HELMET" /* , ... */}

result, err := r.PreRenderSkyBlockItemIDs(context.Background(), ids, renderer.PreRenderOptions{
	Workers:        8,         // 0 means GOMAXPROCS
	Overwrite:      false,     // keep images already in the cache
	ShowProgress:   true,      // progress bar with ETA
	ProgressWriter: os.Stderr, // where the bar goes
})
if err != nil {
	log.Fatal(err)
}
fmt.Printf("%d rendered, %d skipped, %d failed\n", result.Succeeded, result.Skipped, result.Failed)
for _, entry := range result.Entries {
	if entry.Error != "" {
		fmt.Printf("%s: %s\n", entry.InputID, entry.Error)
	}
}
```

Cancel the context to stop early; the images done so far stay in the cache.

## Serving the images

`RenderedItem.Path` is a file in `CacheDir`. Its name lists the item, pack and model it was drawn from, and ends with
a hash of everything that decides the image (textures, pack stack, size), so a path always holds the same picture. Serve the `rendered/` folder as static files with a long
`Cache-Control`, and hand out URLs to it instead of re-rendering per request.

## Lower-level access

`r.MinecraftRenderer()` returns the underlying `MinecraftBlockRenderer` for what the high-level API doesn't cover,
such as resolving which texture and pack an item would use without rendering it, or rendering to an image in memory
instead of the cache.

## Testing

```bash
go test ./...

# Rendering benchmarks; run them before and after performance changes
go test -run '^$' -bench=. .
```
