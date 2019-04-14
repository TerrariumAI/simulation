package vec2

// Vec2 - Hold positions as x and y
type Vec2 struct {
	X int32
	Y int32
}

// GetRegion - Returns the region that a position is in
func (v *Vec2) GetRegion(regionSize int32) Vec2 {
	x := v.X
	y := v.Y
	if x < 0 {
		x -= regionSize
	}
	if y < 0 {
		y -= regionSize
	}
	return Vec2{x / regionSize, y / regionSize}
}

// GetPositionsInRegion - Returns all positions that are in a specfic region
func (v *Vec2) GetPositionsInRegion(regionSize int32) []Vec2 {
	positions := []Vec2{}
	for x := v.X * regionSize; x < v.X*regionSize+regionSize; x++ {
		for y := v.Y * regionSize; y < v.Y*regionSize+regionSize; y++ {
			positions = append(positions, Vec2{x, y})
		}
	}
	return positions
}
