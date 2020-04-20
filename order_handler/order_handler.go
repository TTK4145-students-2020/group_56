package order_handler

import (
	"log"
	"math"

	"../elevator"
	"../elevio"
	"../elevstate"
	"../requests"
)

// se på SystemState.json, cleare gamle unassignedRequests, extract nye ordre
func HandleSystemStateFromMaster(systemState elevstate.System) (newOrders []elevio.ButtonEvent, hallLights [4][2]bool, err error) {
	localState, err := elevstate.RetrieveState()

	if err != nil {
		log.Println(err)
		return
	}

	// Finner master sin versjon av state
	var remoteState elevstate.State
	for _, state := range systemState.States {
		if state.ID == localState.ID {
			remoteState = state
			break
		}
	}

	// Clearer de unassignedRequestene (fra localState) som master har sett
	for _, r1 := range remoteState.NewRequests {
		for i, r2 := range localState.NewRequests {
			if r1.Floor == r2.Floor && r1.Button == r2.Button {
				// Slette ett element på index i
				localState.NewRequests = append(localState.NewRequests[:i], localState.NewRequests[i+1:]...)

				break
			}
		}
	}

	// Extract nye ordre og hallLights
	newOrders = remoteState.NewOrders
	hallLights = systemState.HallLights

	//remoteState.newOrders = []
	//localState.NewOrders = append(localState.NewOrders, newOrders...)
	localState.NewOrders = newOrders

	// Lagre State
	err = elevstate.StateStore(localState)
	// Tenker da ikke å sende state tilbake til master med en gang, etter som det vil gjøres når chaffeur tar imot de nye ordrene
	if err != nil {
		log.Println(err)
	}
	return
}

func updateHallLights(systemState elevstate.System) (hallLights [4][2]bool) {
	for f := 0; f < elevator.NumFloors; f++ {
		btnTypes := []elevio.ButtonType{elevio.BT_HallUp, elevio.BT_HallDown}
		for bt := range btnTypes {
			for _, state := range systemState.States {
				if state.Requests[f][bt] {
					hallLights[f][bt] = true
					break
				}
				hallLights[f][bt] = false
			}
		}
	}
	return
}

// må ta hensysn til Requests, newOrders og unassignedRequests
func GetHallLights() (hallLights [4][2]bool, err error) {
	systemState, err := elevstate.RetrieveSystemState()
	if err != nil {
		log.Println(err)
		return
	}

	// first, requests
	for f := 0; f < elevator.NumFloors; f++ {
		btnTypes := []elevio.ButtonType{elevio.BT_HallUp, elevio.BT_HallDown}
		for bt := range btnTypes {
			for _, state := range systemState.States {
				if state.Requests[f][bt] {
					hallLights[f][bt] = true
					break
				}
			}
		}
	}
	// unassigned og neworders
	for _, state := range systemState.States {
		for _, nre := range state.NewRequests {
			hallLights[nre.Floor][nre.Button] = true
		}
		for _, nor := range state.NewOrders {
			hallLights[nor.Floor][nor.Button] = true
		}
	}

	err = elevstate.SystemStore(systemState)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// ta inn staten til slaven som string (json), cleare newOrder i SystemState.json, se etter unassigned hos alle slaver
func HandleStateFromSlave(slaveState elevstate.State) (unassignedRequests []elevio.ButtonEvent, err error) {
	systemState, err := elevstate.RetrieveSystemState()

	if err != nil {
		log.Println(err)
		return
	}

	// Finner index i system
	systemIndex := -1
	for i, state := range systemState.States {
		if state.ID == slaveState.ID {
			systemIndex = i
			break
		}
	}
	if systemIndex == -1 {
		systemState.States = append(systemState.States, slaveState)
		systemIndex = len(systemState.States) - 1
	}

	// Clearer de newOrder-ene (fra system) som slave har sett
	for _, r1 := range slaveState.NewOrders {
		for i, r2 := range systemState.States[systemIndex].NewOrders {
			if r1.Floor == r2.Floor && r1.Button == r2.Button {
				// Slette ett element på index i
				systemState.States[systemIndex].NewOrders = append(systemState.States[systemIndex].NewOrders[:i], systemState.States[systemIndex].NewOrders[i+1:]...)
				break
			}
		}
	}

	unassignedRequests = slaveState.NewRequests

	systemState.States[systemIndex].NewRequests = unassignedRequests
	systemState.States[systemIndex].Floor = slaveState.Floor
	systemState.States[systemIndex].Dirn = slaveState.Dirn
	systemState.States[systemIndex].Requests = slaveState.Requests

	err = elevstate.SystemStore(systemState)

	if err != nil {
		log.Println(err)
	}
	return
}

// lagre ny request i unassignedRequest
func NewRequest(request elevio.ButtonEvent) (err error) {
	localState, err := elevstate.RetrieveState()

	if err != nil {
		log.Println(err)
		return
	}

	localState.NewRequests = append(localState.NewRequests, request)

	err = elevstate.StateStore(localState)

	if err != nil {
		log.Println(err)
	}
	return
}

func AssignNewOrder(masterID string, newRequest elevio.ButtonEvent) (toMaster bool, err error) {

	systemState, err := elevstate.RetrieveSystemState()

	if err != nil {
		log.Println(err)
	}

	bestIndex := findBestElevIndex(newRequest, systemState.States)

	systemState.States[bestIndex].NewOrders = append(systemState.States[bestIndex].NewOrders, newRequest)

	toMaster = systemState.States[bestIndex].ID == masterID

	err = elevstate.SystemStore(systemState)

	if err != nil {
		log.Println(err)
	}
	return
}

func findBestElevIndex(hallBtn elevio.ButtonEvent, states []elevstate.State) int {
	bestTime := math.MaxInt64
	bestIndex := 0

	for i, state := range states {
		e := elevator.Uninitialized()
		e.Floor = state.Floor
		e.Dirn = elevstate.StringToDir(state.Dirn)
		e.Requests = state.Requests
		e.Requests[hallBtn.Floor][hallBtn.Button] = true

		tti := timeToIdle(e)
		if tti < bestTime {
			bestTime = tti
			bestIndex = i
		}
	}

	return bestIndex //int
}

func timeToIdle(e elevator.Elevator) int {
	TRAVEL_TIME := 2000
	DOOR_OPEN_TIME := e.Config.DoorOpenDuration
	// D_Stop := elevio.MD_Stop

	duration := 0

	switch e.State {
	case elevator.EBIdle:
		e.Dirn = requests.ChooseDirection(e)
		if e.Dirn == elevio.MD_Stop {
			return duration
		}
		break

	case elevator.EBMoving:
		duration += TRAVEL_TIME / 2
		e.Floor += int(e.Dirn)
		break

	case elevator.EBDoorOpen:
		duration -= DOOR_OPEN_TIME / 2
	}

	for {
		if requests.ShouldStop(e) {
			e = requests.ClearAtCurrentFloor(e) // TODO: må gjøre om ClearAtCurrentFloor i  requests.go
			duration += DOOR_OPEN_TIME
			e.Dirn = requests.ChooseDirection(e)
			if e.Dirn == elevio.MD_Stop {
				return duration
			}
		}
		e.Floor += int(e.Dirn)
		duration += TRAVEL_TIME
	}
}
