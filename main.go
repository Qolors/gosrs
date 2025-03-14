package main

import (
	"fmt"
	"time"

	"github.com/qolors/gosrs/internal/services"
)

func main() {

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	done := make(chan struct{})
	badrequests := make(chan struct{}, 1)

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

	<-done
}

func runprocess() error {

	_, err := services.GetPlayerData()

	if err != nil {
		return err
	}

	return nil
}
