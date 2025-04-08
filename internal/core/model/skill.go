package model

type Skill struct {
	ID    int16  `json:"id"`
	Name  string `json:"name"`
	Rank  int32  `json:"rank"`
	Level int32  `json:"level"`
	XP    int64  `json:"xp"`
}
