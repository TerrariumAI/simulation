package main

// Simple struct for holding positions
type Vec2 struct {
	x int32
	y int32
}

func (v *Vec2) GetRegion() Vec2 {
	x := v.x
	y := v.y
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
	if v.x < 0 {
		signX = -1
	}
	if v.y < 0 {
		signY = -1
	}
	startX := (v.x - signX) * region_size
	startY := (v.y - signY) * region_size
	endX := v.x * region_size
	endY := v.y * region_size
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
