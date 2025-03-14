package services

import (
	"github.com/qolors/gosrs/internal/osrsclient"
)

func GetPlayerData() (osrsclient.APIResponse, error) {

	apiData, err := osrsclient.FetchData()

	if err != nil {
		return osrsclient.APIResponse{}, err
	}

	return osrsclient.ConvertToDTO(apiData), nil
}
