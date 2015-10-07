package main

func BoundInt(val int32, lower int32, upper int32) int32 {
	if val > upper {
		return upper
	} else if val < lower {
		return lower
	} else {
		return val
	}
}
