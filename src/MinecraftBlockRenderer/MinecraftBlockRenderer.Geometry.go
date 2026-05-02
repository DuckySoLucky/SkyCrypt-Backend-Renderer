package minecraftblockrenderer

import (
	"duckysolucky/gorenderer/src"
	geometry "duckysolucky/gorenderer/src/Geometry"
	"duckysolucky/gorenderer/src/data"
	"duckysolucky/gorenderer/src/model"
	"image"
)

func (_minecraftBlockRenderer *MinecraftBlockRenderer) BuildTriangles(blockMModel *data.BlockModelInstance, transform model.Matrix4, applyInventoryLighting bool, blockName string) []VisibleTriangle {
	triangles := make([]VisibleTriangle, 0, len(blockMModel.Elements)*12)

	for elementIndex, element := range blockMModel.Elements {
		elementTriangles := _minecraftBlockRenderer.BuildTrianglesForElement(blockMModel, element, transform, elementIndex, applyInventoryLighting, blockName)
		triangles = append(triangles, elementTriangles...)
	}

	return triangles
}
func (_minecraftBlockRenderer *MinecraftBlockRenderer) BuildTrianglesForElement(blockMModel *data.BlockModelInstance, element data.ModelElement, transform model.Matrix4, elementIndex int, applyInventoryLighting bool, blockName string) []VisibleTriangle {
	vertices := BuildElementVertices(element)
	_minecraftBlockRenderer.ApplyElementRotation(element, vertices)
	results := make([]VisibleTriangle, 0, len(element.Faces)*2)

	for direction, face := range element.Faces {
		textureId := ResolveTexture(face.Texture, blockMModel)
		var texture *image.RGBA

		renderPriority := 0
		if face.TintIndex != nil {
			renderPriority = 1
		}

		if face.TintIndex != nil {
			constantTint := _minecraftBlockRenderer.TryGetConstantTint(textureId, &blockName)
			if constantTint != nil {
				texture = _minecraftBlockRenderer._textureRepository.GetTintedTexture(textureId, *constantTint, float64(ConstantTintStrength), float64(1.0))
			} else if biomeKind := _minecraftBlockRenderer.TryGetBiomeTintKind(textureId, blockName); biomeKind != nil {
				texture = _minecraftBlockRenderer.GetBiomeTintedTexture(textureId, *biomeKind)
			} else {
				fallbackTint := _minecraftBlockRenderer.GetColorFromBlockName(blockName)
				if fallbackTint == nil {
					fallbackTint = _minecraftBlockRenderer.GetColorFromBlockName(textureId)
				}
				if fallbackTint != nil {
					texture = _minecraftBlockRenderer._textureRepository.GetTintedTexture(textureId, *fallbackTint, 1, ColorTintBlend)
				} else {
					texture = _minecraftBlockRenderer._textureRepository.GetTexture(textureId)
				}
			}
		} else {
			texture = _minecraftBlockRenderer._textureRepository.GetTexture(textureId)
		}

		faceUv := _minecraftBlockRenderer.GetFaceUv(face, direction, element)

		if face.Rotation == nil {
			n := 0
			face.Rotation = &n
		}

		uvMap := geometry.CreateUvMap(faceUv, *face.Rotation)
		textureRect := _minecraftBlockRenderer.ComputeTextureRectangle(uvMap, texture)

		rectMinU := float32(textureRect.Min.X) / float32(texture.Bounds().Dx())
		rectRangeU := float32(textureRect.Dx()) / float32(texture.Bounds().Dx())
		rectMinV := float32(textureRect.Min.Y) / float32(texture.Bounds().Dy())
		rectRangeV := float32(textureRect.Dy()) / float32(texture.Bounds().Dy())
		for i := 0; i < 4; i++ {
			if rectRangeU > 1e-6 {
				uvMap[i].X = (uvMap[i].X - rectMinU) / rectRangeU
			} else {
				uvMap[i].X = 0
			}
			if rectRangeV > 1e-6 {
				uvMap[i].Y = (uvMap[i].Y - rectMinV) / rectRangeV
			} else {
				uvMap[i].Y = 0
			}
		}

		indices := geometry.FaceVertexIndices[direction]
		localFace := [4]data.Vector3{}
		for i := 0; i < 4; i++ {
			localFace[i] = vertices[indices[i]]
		}

		transformed := [4]data.Vector3{}
		for i := 0; i < 4; i++ {
			transformed[i] = model.Transform(localFace[i], transform)
		}

		depth := (transformed[0].Z + transformed[1].Z + transformed[2].Z + transformed[3].Z) * 0.25
		triangle1Normal := data.Cross(data.Sub(transformed[1], transformed[0]), data.Sub(transformed[2], transformed[0]))
		triangle2Normal := data.Cross(data.Sub(transformed[2], transformed[0]), data.Sub(transformed[3], transformed[0]))
		triangle1Centroid := data.Scale(data.Add(data.Add(transformed[0], transformed[1]), transformed[2]), 1.0/3.0)
		triangle2Centroid := data.Scale(data.Add(data.Add(transformed[0], transformed[2]), transformed[3]), 1.0/3.0)
		shadingEnabled := applyInventoryLighting && element.Shade
		triangle1Shading := float32(1)
		if shadingEnabled {
			triangle1Shading = _minecraftBlockRenderer.ComputeInventoryLightingIntensity(triangle1Normal)
		}
		triangle2Shading := float32(1)
		if shadingEnabled {
			triangle2Shading = _minecraftBlockRenderer.ComputeInventoryLightingIntensity(triangle2Normal)
		}

		results = append(results, VisibleTriangle{
			V1:             transformed[0],
			V2:             transformed[1],
			V3:             transformed[2],
			T1:             src.Vector2{X: uvMap[0].X, Y: uvMap[0].Y},
			T2:             src.Vector2{X: uvMap[1].X, Y: uvMap[1].Y},
			T3:             src.Vector2{X: uvMap[2].X, Y: uvMap[2].Y},
			Texture:        texture,
			TextureRect:    textureRect,
			Depth:          depth,
			Normal:         triangle1Normal,
			Centroid:       triangle1Centroid,
			Direction:      direction,
			ElementIndex:   elementIndex,
			RenderPriority: renderPriority,
			Shading:        triangle1Shading,
		})

		results = append(results, VisibleTriangle{
			V1:             transformed[0],
			V2:             transformed[2],
			V3:             transformed[3],
			T1:             src.Vector2{X: uvMap[0].X, Y: uvMap[0].Y},
			T2:             src.Vector2{X: uvMap[2].X, Y: uvMap[2].Y},
			T3:             src.Vector2{X: uvMap[3].X, Y: uvMap[3].Y},
			Texture:        texture,
			TextureRect:    textureRect,
			Depth:          depth,
			Normal:         triangle2Normal,
			Centroid:       triangle2Centroid,
			Direction:      direction,
			ElementIndex:   elementIndex,
			RenderPriority: renderPriority,
			Shading:        triangle2Shading,
		})
	}

	return results
}

func BuildElementVertices(element data.ModelElement) [8]data.Vector3 {
	min := element.From
	max := element.To

	fx := NormalizeComponent(float64(min.X))
	fy := NormalizeComponent(float64(min.Y))
	fz := NormalizeComponent(float64(min.Z))
	tx := NormalizeComponent(float64(max.X))
	ty := NormalizeComponent(float64(max.Y))
	tz := NormalizeComponent(float64(max.Z))

	return [8]data.Vector3{
		{fx, fy, fz},
		{tx, fy, fz},
		{tx, ty, fz},
		{fx, ty, fz},
		{fx, fy, tz},
		{tx, fy, tz},
		{tx, ty, tz},
		{fx, ty, tz},
	}
}

func NormalizeComponent(value float64) float32 {
	return float32(value/16.0 - 0.5)
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) ApplyElementRotation(element data.ModelElement, vertices [8]data.Vector3) {
	if element.Rotation == nil {
		return
	}

	var axis data.Vector3
	switch element.Rotation.Axis {
	case "x":
		axis = data.Vector3{X: 1, Y: 0, Z: 0}
	case "z":
		axis = data.Vector3{X: 0, Y: 0, Z: 1}
	default:
		axis = data.Vector3{X: 0, Y: 1, Z: 0}
	}

	angle := element.Rotation.AngleInDegrees * DegreesToRadians
	pivot := data.Vector3{
		X: NormalizeComponent(float64(element.Rotation.Origin.X)),
		Y: NormalizeComponent(float64(element.Rotation.Origin.Y)),
		Z: NormalizeComponent(float64(element.Rotation.Origin.Z)),
	}

	rotationMatrix := model.CreateFromAxisAngle(axis, angle)

	for i := range vertices {
		relative := data.Vector3{
			X: vertices[i].X - pivot.X,
			Y: vertices[i].Y - pivot.Y,
			Z: vertices[i].Z - pivot.Z,
		}
		relative = model.Transform(relative, rotationMatrix)
		vertices[i] = data.Vector3{
			X: relative.X + pivot.X,
			Y: relative.Y + pivot.Y,
			Z: relative.Z + pivot.Z,
		}
	}
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) GetFaceUv(face data.ModelFace, direction data.BlockFaceDirection, element data.ModelElement) data.Vector4 {
	if face.Uv != nil {
		return *face.Uv
	}

	return geometry.DefaultFaceUv(element.From, element.To, direction)
}

func (_minecraftBlockRenderer *MinecraftBlockRenderer) ComputeTextureRectangle(uvMap []src.Vector2, texture *image.RGBA) image.Rectangle {
	widthFactor := float32(texture.Bounds().Dx())
	heightFactor := float32(texture.Bounds().Dy())

	minU := uvMap[0].X
	maxU := uvMap[0].X
	minV := uvMap[0].Y
	maxV := uvMap[0].Y

	for i := 1; i < len(uvMap); i++ {
		if uvMap[i].X < minU {
			minU = uvMap[i].X
		}
		if uvMap[i].X > maxU {
			maxU = uvMap[i].X
		}
		if uvMap[i].Y < minV {
			minV = uvMap[i].Y
		}
		if uvMap[i].Y > maxV {
			maxV = uvMap[i].Y
		}
	}

	minX := int(float32(minU)*widthFactor + 0.5)
	maxX := int(float32(maxU)*widthFactor + 0.5)
	minY := int(float32(minV)*heightFactor + 0.5)
	maxY := int(float32(maxV)*heightFactor + 0.5)

	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX < minX+1 {
		maxX = minX + 1
	}
	if maxY < minY+1 {
		maxY = minY + 1
	}
	if maxX > texture.Bounds().Dx() {
		maxX = texture.Bounds().Dx()
	}
	if maxY > texture.Bounds().Dy() {
		maxY = texture.Bounds().Dy()
	}

	return image.Rect(minX, minY, maxX, maxY)
}
