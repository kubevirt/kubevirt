package domainstats

func nanosecondsToSeconds(ns uint64) float64 {
	return float64(ns) / 1000000000
}

func kibibytesToBytes(kibibytes uint64) float64 {
	return float64(kibibytes) * 1024
}
