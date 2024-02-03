package util

import "math"

func AbsInt(x int) int {
	return AbsDiffInt(x, 0)
}

func AbsDiffInt(x int, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}

/** Cantor pairing function adapted to work with negatives */
func Pair(i int, j int) uint {
	// TODO: implement checks on overflow limits
	return cantorPair(intToNat(i), intToNat(j))
}

/** Inverts the Cantor pairing including back to negative numbers */
func InvertPair(z uint) (int, int) {
	// TODO: implement checks on overflow limits
	x, y := invertCantorPair(z)
	return natToInt(x), natToInt(y)
}

func intToNat(i int) uint {
	if i < 0 {
		return uint(-2*i - 1)
	}
	return uint(2 * i)
}

func natToInt(i uint) int {
	if i%2 == 0 {
		return int(i / 2)
	}
	return -int((i + 1) / 2)
}

func cantorPair(i uint, j uint) uint {
	return (i+j)*(i+j+1)/2 + j
}

func invertCantorPair(z uint) (uint, uint) {
	w := math.Abs(0.5 * (math.Sqrt(8*float64(z)) - 1))
	t := 0.5 * (w*w + w)
	y := float64(z) - t
	x := w - y
	return uint(x + 0.5), uint(y + 0.5)
}
