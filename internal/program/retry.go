package program

// Attempts is how many times a step may run, including the first try.
// Global MaxAttempts > 0 is a hard cap. Global 0 is no cap. Recipe 0
// means unset: use global when Always is set, else one try.
func Attempts(gAlways bool, gMax int, lAlways bool, lMax int) int {
	n := 1
	if lMax > 0 {
		n = lMax
	} else if (lAlways || gAlways) && gMax > 0 {
		n = gMax
	}
	if gMax > 0 && n > gMax {
		n = gMax
	}
	return max(n, 1)
}

func Always(gAlways, lAlways bool) bool {
	return gAlways || lAlways
}
