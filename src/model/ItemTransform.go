package model

import (
	"math"

	"duckysolucky/gorenderer/src/data"
)

type Matrix4 [4][4]float32

func IdentityMatrix() Matrix4 {
	var m Matrix4
	for i := 0; i < 4; i++ {
		m[i][i] = 1
	}
	return m
}

func MulMatrix(a, b Matrix4) Matrix4 {
	var r Matrix4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[i][k] * b[k][j]
			}
			r[i][j] = s
		}
	}
	return r
}

func CreateTranslation(tx, ty, tz float32) Matrix4 {
	m := IdentityMatrix()
	m[0][3] = tx
	m[1][3] = ty
	m[2][3] = tz
	return m
}

func CreateScale(s data.Vector3) Matrix4 {
	m := IdentityMatrix()
	m[0][0] = s.X
	m[1][1] = s.Y
	m[2][2] = s.Z
	return m
}

func CreateScaleWithFloat(s float32) Matrix4 {
	m := IdentityMatrix()
	m[0][0] = s
	m[1][1] = s
	m[2][2] = s
	return m
}

func CreateRotationX(rad float32) Matrix4 {
	m := IdentityMatrix()
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))
	m[1][1] = c
	m[1][2] = -s
	m[2][1] = s
	m[2][2] = c
	return m
}

func CreateRotationY(rad float32) Matrix4 {
	m := IdentityMatrix()
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))
	m[0][0] = c
	m[0][2] = s
	m[2][0] = -s
	m[2][2] = c
	return m
}

func CreateRotationZ(rad float32) Matrix4 {
	m := IdentityMatrix()
	c := float32(math.Cos(float64(rad)))
	s := float32(math.Sin(float64(rad)))
	m[0][0] = c
	m[0][1] = -s
	m[1][0] = s
	m[1][1] = c
	return m
}

type ItemTransform struct {
	Rotation    data.Vector3
	Translation data.Vector3
	Scale       data.Vector3
}

var NoTransform = ItemTransform{Rotation: data.Vector3{0, 0, 0}, Translation: data.Vector3{0, 0, 0}, Scale: data.Vector3{1, 1, 1}}

func (t ItemTransform) equals(o ItemTransform) bool {
	return t.Rotation == o.Rotation && t.Translation == o.Translation && t.Scale == o.Scale
}

// BuildMatrix returns a 4x4 transformation matrix equivalent to the C# implementation.
func (t ItemTransform) BuildMatrix(isLeftHand bool) Matrix4 {
	if t.equals(NoTransform) {
		return IdentityMatrix()
	}

	translationX := t.Translation.X
	rotationY := t.Rotation.Y
	rotationZ := t.Rotation.Z
	if isLeftHand {
		translationX = -translationX
		rotationY = -rotationY
		rotationZ = -rotationZ
	}

	translationMatrix := CreateTranslation(translationX/16.0, t.Translation.Y/16.0, t.Translation.Z/16.0)

	degToRad := float32(math.Pi / 180.0)
	rotationMatrix := MulMatrix(CreateRotationZ(rotationZ*degToRad), MulMatrix(CreateRotationY(rotationY*degToRad), CreateRotationX(t.Rotation.X*degToRad)))

	scaleMatrix := CreateScale(t.Scale)

	// scale * rotation * translation
	return MulMatrix(scaleMatrix, MulMatrix(rotationMatrix, translationMatrix))
}

func MultiplyMatrix(a, b Matrix4) Matrix4 {
	var r Matrix4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[i][k] * b[k][j]
			}
			r[i][j] = s
		}
	}
	return r
}

func CreateFromAxisAngle(axis data.Vector3, angle float32) Matrix4 {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	t := 1 - c

	x := axis.X
	y := axis.Y
	z := axis.Z

	m := IdentityMatrix()
	m[0][0] = t*x*x + c
	m[0][1] = t*x*y - s*z
	m[0][2] = t*x*z + s*y

	m[1][0] = t*x*y + s*z
	m[1][1] = t*y*y + c
	m[1][2] = t*y*z - s*x

	m[2][0] = t*x*z - s*y
	m[2][1] = t*y*z + s*x
	m[2][2] = t*z*z + c

	return m
}

func Transform(relative data.Vector3, rotationMatrix Matrix4) data.Vector3 {
	x := rotationMatrix[0][0]*relative.X + rotationMatrix[0][1]*relative.Y + rotationMatrix[0][2]*relative.Z + rotationMatrix[0][3]
	y := rotationMatrix[1][0]*relative.X + rotationMatrix[1][1]*relative.Y + rotationMatrix[1][2]*relative.Z + rotationMatrix[1][3]
	z := rotationMatrix[2][0]*relative.X + rotationMatrix[2][1]*relative.Y + rotationMatrix[2][2]*relative.Z + rotationMatrix[2][3]
	return data.Vector3{X: x, Y: y, Z: z}
}
