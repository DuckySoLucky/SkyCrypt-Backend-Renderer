# Triangle Output Comparison: C# vs Go

## Critical Findings

### 1. **Coordinate System Inversion** ⚠️

**C# Triangle 0:**
```
V1=<0.44194174,  0.27063292,  0.15625003>
V2=<4.4703484E-08, 0.4916038,  -0.22648275>
V3=<-0.44194174, 0.27063295,  0.15624999>
```

**Go Triangle 0:**
```
V1={0.4419417382415922  -0.27063293868263716  0.1562499999999999}
V2={8.326672684688674e-17  -0.04966206956184102  0.5389827723098716}
V3={-0.4419417382415922  -0.27063293868263705  0.15625000000000003}
```

**Pattern:** Y and Z coordinates are **inverted/swapped/negated** between C# and Go.

| Axis | C# | Go | Relationship |
|------|----|----|--------------|
| X | ✓ Same | ✓ Same | Matches |
| Y | +0.27 | -0.27 | **NEGATED** |
| Z | -0.226 | +0.538 | **DIFFERENT** |

---

### 2. **Triangle Ordering Mismatch**

**C# Triangles 0-1:** ElementIndex=0, Direction=unknown
**Go Triangles 0-1:** ElementIndex=0, Direction=5 (South)

**C# Triangles 2-3:** ElementIndex=0, Direction=unknown (from Normal pattern)
**Go Triangles 2-3:** ElementIndex=0, Direction=0 (Down)

The **same geometric triangles are being paired differently** or **Direction enum values differ**.

---

### 3. **Normal Vector Discrepancies** ⚠️

**C# Triangle 0 Normal:**
```
Normal=<1.528158E-09, 0.33829114, 0.19531249>
```

**Go Triangle 0 Normal:**
```
Normal={-1.1826005276036221e-17 -0.3382911733532964 0.19531249999999997}
```

| Component | C# | Go | Difference |
|-----------|----|----|------------|
| X | ≈0 | ≈0 | ✓ Negligible |
| Y | +0.3383 | **-0.3383** | **SIGN FLIPPED** |
| Z | +0.1953 | +0.1953 | ✓ Same |

**Impact:** Lighting/culling logic based on normal dot products will fail.

---

### 4. **Shading Factor Completely Wrong** ⚠️

| Triangle | C# Shading | Go Shading | Delta |
|----------|-----------|-----------|-------|
| 0 | 1.0 | 0.862887 | **-0.137113** |
| 1 | 1.0 | 0.862887 | **-0.137113** |
| 2 | 0.60047233 | 0.627014 | **+0.026512** |
| 3 | 0.60047233 | 0.627014 | **+0.026512** |
| 4 | 0.89243 | 0.335056 | **-0.557374** |
| 5 | 0.89243007 | 0.335056 | **-0.557374** |

**This is NOT a floating-point precision issue.** Shading values differ by **5-55%**.

---

## Root Cause Analysis

### Hypothesis 1: Coordinate System Handedness
- **Left-handed vs Right-handed** coordinate systems
- Typically affects Y and Z axes
- Would explain Y negation and Z reordering

### Hypothesis 2: Screen Space vs World Space
- Y-axis inversion is common when converting from world space (up=+Y) to screen space (down=+Y)
- Z-axis inversion is common when converting from right-handed (forward=-Z) to left-handed (forward=+Z)

### Hypothesis 3: Transform Matrix Error
The transform matrix applied to vertices may have:
- Incorrect rotation or reflection
- Axis swaps
- Sign errors in specific components

### Hypothesis 4: Lighting Calculation Bug
The `ComputeInventoryLightingIntensity()` function receives **inverted normals**, which would cause completely different shading values.

---

## Specific Code Issues to Investigate

### In MinecraftBlockRenderer.Rendering.go:

1. **Line 131-134** - Triangle debug output: Check if coordinates are pre-projection or post-projection
2. **Line 25-26** - InventoryLightDirection: Verify this matches C#
3. **`ComputeInventoryLightingIntensity()`** function: The normal calculation may be receiving inverted input

### Potential Fixes:

```go
// Option 1: Negate Y in BuildTriangles
transformed[i] = model.Transform(localFace[i], transform)
transformed[i].Y = -transformed[i].Y  // Screen space inversion

// Option 2: Negate Normal components after cross product
triangle1Normal := data.Cross(data.Sub(transformed[1], transformed[0]), data.Sub(transformed[2], transformed[0]))
triangle1Normal.Y = -triangle1Normal.Y  // If normal inversion is needed
```

---

## Verification Checklist

- [ ] Compare transform matrix generation between C# and Go
- [ ] Verify coordinate system (left-handed vs right-handed) in both implementations
- [ ] Check if Y-axis inversion should occur before or after projection
- [ ] Verify InventoryLightDirection vector matches between implementations
- [ ] Test with identity transform to isolate the issue
- [ ] Compare raw model vertices before any transformation
- [ ] Check if `ApplyElementRotation()` handles axes correctly in Go vs C#

---

## Summary

**Status:** ❌ **NOT MATCHING**

| Issue | Severity | Type |
|-------|----------|------|
| Y-coordinate negation | Critical | Geometry |
| Z-coordinate mismatch | Critical | Geometry |
| Normal Y-component sign flip | Critical | Lighting |
| Shading factor 5-55% off | Critical | Illumination |
| Triangle ordering discrepancy | High | Metadata |

**Recommendation:** Investigate the transform matrix building logic and coordinate system conventions before lighting calculations.
