package osrsclient

import (
	"encoding/json"
	"io"
	"net/http"
)

func FetchData() (apiResponseModel, error) {

	resp, err := http.Get("https://secure.runescape.com/m=hiscore_oldschool/index_lite.json?player=An%20Okay%20Time")

	if err != nil {
		return apiResponseModel{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return apiResponseModel{}, err
	}

	var apiModel apiResponseModel

	jsonerr := json.Unmarshal(body, &apiModel)

	if jsonerr != nil {
		return apiResponseModel{}, jsonerr
	}

	return apiModel, nil

}
