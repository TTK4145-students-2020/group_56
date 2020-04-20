package chaffeur

import (
	"time"

	"../elevio"
	"../fsm"
	"../timer"
)

func Chaffeur(event_localRequest chan<- elevio.ButtonEvent, event_stateChange chan<- struct{}, drv_order <-chan elevio.ButtonEvent, drv_hallLights <-chan [4][2]bool, port string) {
	numFloors := 4

	elevio.Init(port, numFloors)

	drv_floors := make(chan int)
	drv_timeout := make(chan bool)

	go elevio.PollButtons(event_localRequest)

	go elevio.PollFloorSensor(drv_floors)
	go timer.PollTimer(drv_timeout)

	fsm.OnInitBetweenFloors()

	for {
		select {

		case a := <-drv_order:
			fsm.OnNewRequest(a.Floor, a.Button)

		case a := <-drv_floors:
			fsm.OnFloorArrival(a)

		case a := <-drv_timeout:
			if a {
				fsm.OnDoorTimeout()
				timer.Stop()
			}

		case a := <-drv_hallLights:
			fsm.SetHallLights(a)

		}

		fsm.TransmitState()
		go func(){event_stateChange <- struct{}{}}()

		time.Sleep(20 * time.Millisecond)
	}
}
