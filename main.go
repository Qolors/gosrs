package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/qolors/gosrs/internal/services"
)

func main() {

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	courierTicker := time.NewTicker(15 * time.Minute)
	defer courierTicker.Stop()

	services.InitDBService(context.Background())
	defer services.CloseConnection()

	courier := services.NewCourier()
	defer courier.Stop()

	done := make(chan struct{})
	badrequests := make(chan struct{}, 1)

	fmt.Println("Starting Loop..")

	courier.Hitch(courierTicker)

	go func() {

		badRequestCount := 0

		for {
			select {
			case <-ticker.C:
				fmt.Println("Running Poll at", time.Now())
				if err := runprocess(); err != nil {
					badRequestCount++
					fmt.Print("Job Fail")
					if badRequestCount >= 3 {
						fmt.Println("Exiting as too many Bad Requests..")
						badrequests <- struct{}{}
						return
					}
				}
				fmt.Print("Job Success")
			}
		}
	}()

	go func() {
		<-badrequests
		fmt.Println("Exiting Program..")
		close(done)
	}()

	courier.Start()

	<-done
}

func runprocess() error {

	apiResponse, err := services.GetPlayerData()

	if err != nil {
		return err
	}

	activities, err := json.Marshal(apiResponse.Activities)
	skills, err := json.Marshal(apiResponse.Skills)

	if err != nil {
		return err
	}

	err = services.InsertPolling(skills, activities)

	return err
}
