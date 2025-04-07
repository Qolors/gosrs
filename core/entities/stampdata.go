package entities

import "time"

type StampedData struct {
	Activities []Activity
	Skills     []Skill
	Timestamp  time.Time
}
