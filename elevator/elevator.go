package elevator

import (
	"../elevio"
)

const NumFloors int = 4
const NumButtons int = 3

type motorState int

const (
	EBIdle     motorState = 0
	EBDoorOpen            = 1
	EBMoving              = 2
)

type ClearRequestVariant int

const (
	CVALL    ClearRequestVariant = 0
	CVInDirn                     = 1
)

type Elevator struct {
	Floor    int
	Dirn     elevio.MotorDirection
	Requests [NumFloors][NumButtons]bool
	State    motorState

	Config struct {
		ClearRequestVariant ClearRequestVariant
		DoorOpenDuration    int
	}
}

func Uninitialized() (elevator Elevator) {
	elevator.Floor = -1
	elevator.Dirn = elevio.MD_Stop
	elevator.State = EBIdle
	elevator.Config.ClearRequestVariant = CVALL
	elevator.Config.DoorOpenDuration = 3
	return
}
