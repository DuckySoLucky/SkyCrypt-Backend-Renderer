package main

import (
	minecraftblockrenderer "duckysolucky/gorenderer/src/MinecraftBlockRenderer"
	"fmt"
	"image/png"
	"os"
)

func main() {
	// if len(os.Args) < 2 {
	// 	fmt.Println("Usage: go run main.go <assets_directory>")
	// 	return
	// }

	// assetsPath := os.Args[1]
	// resolvedPath, err := assets.ResolveAssetsDirectory(assetsPath)
	// if err != nil {
	// 	fmt.Printf("Error resolving assets directory: %v\n", err)
	// 	return
	// }

	// fmt.Printf("Assets directory resolved to: %s\n", *resolvedPath)

	// if err := assets.DownloadAssets("1.21.11", *resolvedPath, false); err != nil {
	// 	fmt.Printf("Error downloading assets: %v\n", err)
	// 	return
	// }

	assetsPath := "packs/assets"

	renderer := minecraftblockrenderer.CreateFromMinecraftAssets(assetsPath, nil, nil)

	// stone := renderer.RenderBlock("stone", minecraftblockrenderer.BlockRenderOptions{Size: 256})

	dirt := renderer.RenderGuiItemInternal("birch_chest_boat", &minecraftblockrenderer.BlockRenderOptions{Size: 256}, nil)
	if dirt != nil {
		outputFile, err := os.Create("dirt.png")
		if err != nil {
			fmt.Printf("Error creating output file: %v\n", err)
			return
		}
		defer outputFile.Close()

		if err := png.Encode(outputFile, dirt); err != nil {
			fmt.Printf("Error encoding PNG: %v\n", err)
			return
		}

		fmt.Println("Dirt item rendered and saved as dirt.png")

	}

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

}
