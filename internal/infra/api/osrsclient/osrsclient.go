package osrsclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/qolors/gosrs/internal/core/model"
)

type OSRSClient interface {
	GetPlayerData() (model.StampedData, error)
}

type OSRSClientImpl struct {
	PlayerUrl string
}

func NewOSRSClient(pn string) *OSRSClientImpl {
	formatted_name := strings.Replace(pn, " ", "%20", -1)
	url := fmt.Sprintf("https://secure.runescape.com/m=hiscore_oldschool_ultimate/index_lite.json?player=%s", formatted_name)
	return &OSRSClientImpl{PlayerUrl: url}
}

func (oc *OSRSClientImpl) fetchData() (apiResponseModel, error) {

	resp, err := http.Get(oc.PlayerUrl)

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

func (oc *OSRSClientImpl) GetPlayerData() (model.StampedData, error) {

	apiData, err := oc.fetchData()

	if err != nil {
		return model.StampedData{}, err
	}

	cleaned := ConvertToDTO(apiData)
	timestamp := time.Now().UTC()

	data := &model.StampedData{
		Activities: cleaned.Activities,
		Skills:     cleaned.Skills,
		Timestamp:  timestamp,
	}
	return *data, nil
}
