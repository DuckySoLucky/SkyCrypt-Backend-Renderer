package data

type BlockFaceDirection int

const (
	North BlockFaceDirection = iota
	South
	East
	West
	Up
	Down
)

type Vector3 struct {
	X float32
	Y float32
	Z float32
}

func Cross(a, b Vector3) Vector3 {
	return Vector3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func Sub(a, b Vector3) Vector3 {
	return Vector3{
		X: a.X - b.X,
		Y: a.Y - b.Y,
		Z: a.Z - b.Z,
	}
}

func Add(a, b Vector3) Vector3 {
	return Vector3{
		X: a.X + b.X,
		Y: a.Y + b.Y,
		Z: a.Z + b.Z,
	}
}

func Scale(v Vector3, s float32) Vector3 {
	return Vector3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

type Vector4 struct {
	X float32
	Y float32
	Z float32
	W float32
}

type BlockModelInstance struct {
	Name        string
	ParentChain []string
	Textures    map[string]string
	Display     map[string]*TransformDefinition
	Elements    []ModelElement
}

func (b *BlockModelInstance) GetDisplayTransform(name string) *TransformDefinition {
	if b == nil || b.Display == nil {
		return nil
	}
	return b.Display[name]
}

type ModelElement struct {
	From     Vector3
	To       Vector3
	Rotation *ElementRotation
	Faces    map[BlockFaceDirection]ModelFace
	Shade    bool
}

type ElementRotation struct {
	AngleInDegrees float32
	Origin         Vector3
	Axis           string
	Rescale        bool
}

type ModelFace struct {
	Texture   string
	Uv        *Vector4
	Rotation  *int
	TintIndex *int
	CullFace  *string
}
