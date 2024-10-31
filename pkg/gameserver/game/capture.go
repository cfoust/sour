package game

type CaptureMode interface {
	TeamMode
	Bases(*Team) []int32
}

type captureMode struct {
	teamMode
	bases map[*Team][]int32
}

func (cm *captureMode) Bases(t *Team) []int32 {
	if bases, ok := cm.bases[t]; ok {
		return bases
	}
	return nil
}
