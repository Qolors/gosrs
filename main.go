package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/qolors/gosrs/internal/server"
	"github.com/qolors/gosrs/internal/services"
)

func main() {

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	done := make(chan struct{})
	badrequests := make(chan struct{}, 1)

	fmt.Println("Starting Loop..")

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

	server.StartAndListen()
	defer server.CloseServer()

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

	return err
}
