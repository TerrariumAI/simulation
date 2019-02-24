package main

// Simple struct for holding positions
type Vec2 struct {
	X int32
	Y int32
}

func (v *Vec2) GetRegion() Vec2 {
	x := v.X
	y := v.Y
	var signX int32 = 1
	var signY int32 = 1
	if x < 0 {
		signX = -1
	}
	if y < 0 {
		signY = -1
	}
	return Vec2{x/10 + signX, y/10 + signY}
}

func (v *Vec2) GetPositionsInRegion() ([]int32, []int32) {
	xs := []int32{}
	ys := []int32{}
	var signX int32 = 1
	var signY int32 = 1
	if v.X < 0 {
		signX = -1
	}
	if v.Y < 0 {
		signY = -1
	}
	startX := (v.X - signX) * region_size
	startY := (v.Y - signY) * region_size
	endX := v.X * region_size
	endY := v.Y * region_size
	if signX > 0 {
		for x := startX; x < endX; x++ {
			xs = append(xs, x)
		}
	} else {
		for x := startX; x > endX; x-- {
			xs = append(xs, x)
		}
	}
	if signY > 0 {
		for y := startY; y < endY; y++ {
			ys = append(ys, y)
		}
	} else {
		for y := startY; y > endY; y-- {
			ys = append(ys, y)
		}
	}

	return xs, ys
}
