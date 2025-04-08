package osrsclient

import "github.com/qolors/gosrs/internal/core/model"

func ConvertToDTO(raw apiResponseModel) APIResponse {

	for i := range raw.Activities {

		if raw.Activities[i].Rank == -1 {
			raw.Activities[i].Rank = 0
			raw.Activities[i].Score = 0
		}
	}

	return APIResponse(raw)
}

type APIResponse struct {
	Skills     []model.Skill
	Activities []model.Activity
}

type apiResponseModel struct {
	Skills     []model.Skill    `json:"skills"`
	Activities []model.Activity `json:"activities"`
}
