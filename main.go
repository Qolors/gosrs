package main

import (
	"fmt"
	"time"

	"github.com/qolors/gosrs/internal/services"
	"github.com/qolors/gosrs/internal/services/queue"
)

var pollBuffer *queue.RingBuffer
var c *services.Courier

func main() {

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	done := make(chan struct{})
	badrequests := make(chan struct{}, 1)

	c = services.NewCourier()

	pollBuffer = queue.NewRingBuffer(1440)

	fmt.Println("Starting Loop..")

	go func() {

		badRequestCount := 0

		for {
			select {
			case <-ticker.C:
				fmt.Println("Running Poll at", time.Now().UTC())
				if err := runprocess(); err != nil {
					badRequestCount++
					fmt.Println("Job Fail")
					if badRequestCount >= 3 {
						fmt.Println("Exiting as too many Bad Requests..")
						badrequests <- struct{}{}
						return
					}
				} else {
					fmt.Println("Successful Run")
					badRequestCount = 0
				}
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

	apiResponse, err := services.GetPlayerData()

	if err != nil {
		return err
	}

	stamped := queue.StampedData{Skills: apiResponse.Skills, Activities: apiResponse.Activities, Timestamp: time.Now().UTC()}

	haschange := pollBuffer.Add(stamped)

	if haschange {
		if !c.Running {
			fmt.Println("Starting new xp session..")
			c.Start()
			c.Pack <- stamped
		} else {
			c.Pack <- stamped
		}

	} else {
		if c.Running {
			fmt.Println("Session ending due to 0 change..")
			c.Send <- pollBuffer.GetAll()
		}
	}

	return err
}
