package main

import (
	minecraftblockrenderer "duckysolucky/gorenderer/src/MinecraftBlockRenderer"
	texturepacks "duckysolucky/gorenderer/src/TexturePacks"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	nbt "duckysolucky/gorenderer/src/NBT"
)

func main() {
	// var assetsPath = System.IO.Path.Combine(Directory.GetCurrentDirectory(), "minecraft", "assets", "minecraft");

	// assetsPath := rootPath of the resource pack, which is the same as the path used to create the registry
	cwd, _ := os.Getwd()
	assetsPath := filepath.Join(cwd, "packs", "assets", "minecraft")
	texturePacksPath := filepath.Join(cwd, "texturepacks")

	registry := texturepacks.NewTexturePackRegistry()
	registry.RegisterAllPacks(texturePacksPath, false)
	renderer := minecraftblockrenderer.CreateFromMinecraftAssets(assetsPath, registry, nil)
	renderer.PreloadRegisteredPacks(true)

	packs := renderer.GetLoadedResourcePacks()
	for _, pack := range packs {
		fmt.Printf("Loaded resource pack: %s - (%+v)\n", pack.Pack.DisplayName, pack.Meta.Version)
	}

	// stone := renderer.RenderBlock("stone", minecraftblockrenderer.BlockRenderOptions{Size: 256})

	// dirt := renderer.RenderBlock("crafting_table", minecraftblockrenderer.BlockRenderOptions{Size: 256, EnableAntiAliasing: false})
	// if dirt != nil {
	// 	outputFile, err := os.Create("dirt.png")
	// 	if err != nil {
	// 		fmt.Printf("Error creating output file: %v\n", err)
	// 		return
	// 	}
	// 	defer outputFile.Close()

	// 	if err := png.Encode(outputFile, dirt); err != nil {
	// 		fmt.Printf("Error encoding PNG: %v\n", err)
	// 		return
	// 	}

	// 	fmt.Println("Dirt item rendered and saved as dirt.png")
	// }
	// diamondSword := renderer.RenderItem(
	// 	"minecraft:diamond_sword",
	// 	nil,
	// 	&minecraftblockrenderer.BlockRenderOptions{Size: 256, EnableAntiAliasing: false},
	// )
	// if diamondSword != nil {
	// 	outputFile, err := os.Create("diamond_sword.png")
	// 	if err != nil {
	// 		fmt.Printf("Error creating output file: %v\n", err)
	// 		return
	// 	}
	// 	defer outputFile.Close()

	// 	if err := png.Encode(outputFile, diamondSword); err != nil {
	// 		fmt.Printf("Error encoding PNG: %v\n", err)
	// 		return
	// 	}

	// 	fmt.Println("Diamond sword item rendered and saved as diamond_sword.png")
	// }

	// if stone != nil {
	// 	outputFile, err := os.Create("stone.png")
	// 	if err != nil {
	// 		fmt.Printf("Error creating output file: %v\n", err)
	// 		return
	// 	}
	// 	defer outputFile.Close()

	// 	if err := png.Encode(outputFile, stone); err != nil {
	// 		fmt.Printf("Error encoding PNG: %v\n", err)
	// 		return
	// 	}

	// 	fmt.Println("Stone block rendered and saved as stone.png")

	// }

	// 	var newItemData = new MinecraftBlockRenderer.ItemRenderData(
	// 	CustomData: new NbtCompound(new[]{
	// 		new KeyValuePair<string, NbtTag>("id", new NbtString("ASPECT_OF_THE_VOID"))
	// 	})
	// );

	// var newOptions = MinecraftBlockRenderer.BlockRenderOptions.Default with { PackIds = new[] { "fsr" }, ItemData = newItemData };

	// var output = renderer.RenderGuiItemWithResourceId("minecraft:diamond_sword", newOptions);
	// output.Image.Save("rendered_diamond_sword.png");
	// Console.WriteLine("Saved rendered_diamond_sword.png");
	newItemData := &minecraftblockrenderer.ItemRenderData{
		CustomData: nbt.NewNbtCompound(map[string]nbt.NbtTag{
			"id": nbt.NewNbtString("ASPECT_OF_THE_VOID"),
		}),
	}
	newOptions := minecraftblockrenderer.BlockRenderOptions{
		PackIds:  []string{"fsr"},
		ItemData: newItemData,
	}
	output := renderer.RenderGuiItemWithResourceId("minecraft:diamond_sword", &newOptions)
	if output != nil {
		outputFile, err := os.Create("rendered_diamond_sword_with_resource_id.png")
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			return
		}
		defer outputFile.Close()

		if err := png.Encode(outputFile, output.Image); err != nil {
			fmt.Printf("Error encoding PNG: %v\n", err)
			return
		}

		fmt.Println("Diamond sword with resource ID rendered and saved as rendered_diamond_sword_with_resource_id.png")
	}

}
