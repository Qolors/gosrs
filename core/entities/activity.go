package entities

type Activity struct {
	ID    int16  `json:"id"`
	Name  string `json:"name"`
	Rank  int32  `json:"rank"`
	Score int32  `json:"score"`
}
