package osrsclient

func ConvertToDTO(raw apiResponseModel) APIResponse {

	for i := range raw.Activities {

		if raw.Activities[i].Rank == -1 {
			raw.Activities[i].Rank = 0
			raw.Activities[i].Score = 0
		}
	}

	return APIResponse{
		Skills:     raw.Skills,
		Activities: raw.Activities,
	}
}
