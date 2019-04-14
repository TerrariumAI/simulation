package vec2

import "testing"

const (
	regionSize = 16
)

func TestGetRegion(t *testing.T) {
	// Vectors to test
	vectors := [12]Vec2{
		Vec2{0, 0},
		Vec2{1, 0},
		Vec2{0, 1},
		Vec2{1, 1},
		Vec2{-1, -1},
		Vec2{-1, 0},
		Vec2{0, -1},
		Vec2{regionSize, 0},
		Vec2{0, regionSize},
		Vec2{-regionSize, 0},
		Vec2{regionSize, regionSize},
		Vec2{-regionSize, -regionSize},
	}
	// Expected regions
	expectedRegions := [12]Vec2{
		Vec2{0, 0},
		Vec2{0, 0},
		Vec2{0, 0},
		Vec2{0, 0},
		Vec2{-1, -1},
		Vec2{-1, 0},
		Vec2{0, -1},
		Vec2{1, 0},
		Vec2{0, 1},
		Vec2{-2, 0},
		Vec2{1, 1},
		Vec2{-2, -2},
	}

	for i, vector := range vectors {
		region := vector.getRegion(regionSize)
		expectedRegion := expectedRegions[i]
		if region != expectedRegion {
			t.Errorf("Region for Vect2 %v was %v, expected %v", vector, region, expectedRegion)
		}
	}
}

func TestGetPositionsInRegion(t *testing.T) {
	position := Vec2{0, 0}

	positions := position.getPositionsInRegion(regionSize)
	expectedPositions := []Vec2{}
	for x := 0; x < regionSize; x++ {
		for y := 0; y < regionSize; y++ {
			expectedPositions = append(expectedPositions, Vec2{int32(x), int32(y)})
		}
	}

	if len(positions) != len(expectedPositions) {
		t.Errorf("Length of positions in region is %v, expected %v", len(positions), len(expectedPositions))
	}
	for i, pos := range positions {
		expectedPos := expectedPositions[i]
		if pos != expectedPos {
			t.Errorf("Position in region at index [%v] was %v, expected %v", i, pos, expectedPos)
		}
	}
}
