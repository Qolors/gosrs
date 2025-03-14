package osrsclient

type Skill struct {
	ID    int16  `json:"id"`
	Name  string `json:"name"`
	Rank  int32  `json:"rank"`
	Level int32  `json:"level"`
	XP    int32  `json:"xp"`
}

type Activity struct {
	ID    int16  `json:"id"`
	Name  string `json:"name"`
	Rank  int32  `json:"rank"`
	Score int32  `json:"score"`
}

//keeping separation in prep for possible changes

type APIResponse struct {
	Skills     []Skill
	Activities []Activity
}

type apiResponseModel struct {
	Skills     []Skill    `json:"skills"`
	Activities []Activity `json:"activities"`
}
