package fsm

import (
	//"fmt"

	"../elevator"
	"../elevio"
	"../elevstate"
	"../requests"
	"../timer"
)

var elev elevator.Elevator

func FsmInit() {
	elev = elevator.Uninitialized()
}

/* func setAllLights(e elevator.Elevator) {
	for f := 0; f < elevator.NumFloors; f++ {
		for btn := 0; btn < elevator.NumButtons; btn++ {
			elevio.SetButtonLamp(elevio.ButtonType(btn), f, e.Requests[f][btn])
		}
	}
} */

func setCabLights(e elevator.Elevator) {
	for f := 0; f < elevator.NumFloors; f++ {
		elevio.SetButtonLamp(elevio.BT_Cab, f, e.Requests[f][2])
	}
}

func SetHallLights(boolHallLights [4][2]bool) {
	for f := 0; f < elevator.NumFloors; f++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, f, boolHallLights[f][0])
		elevio.SetButtonLamp(elevio.BT_HallDown, f, boolHallLights[f][1])
	}
}

func SetOneLight(btnType int, floor int) {

	elevio.SetButtonLamp(btnType, floor, true)

}

func OnInitBetweenFloors() {
	elevio.SetMotorDirection(elevio.MD_Down)
	elev.Dirn = elevio.MD_Down
	elev.State = elevator.EBMoving
}

func OnNewRequest(btnFloor int, btnType elevio.ButtonType) {
	switch elev.State {

	case elevator.EBDoorOpen:
		//fmt.Println("Button pressed, Door open")
		if elev.Floor == btnFloor {
			timer.Start(elev.Config.DoorOpenDuration)
		} else {
			elev.Requests[btnFloor][btnType] = true
		}
		break

	case elevator.EBMoving:
		//fmt.Println("Button pressed, Elevator Moving")
		elev.Requests[btnFloor][btnType] = true
		break

	case elevator.EBIdle:
		//fmt.Println("Button pressed, Elevator Idle")
		if elev.Floor == btnFloor {
			elevio.SetDoorOpenLamp(true)
			timer.Start(elev.Config.DoorOpenDuration)
			elev.State = elevator.EBDoorOpen
		} else {
			elev.Requests[btnFloor][btnType] = true
			elev.Dirn = requests.ChooseDirection(elev)
			elevio.SetMotorDirection(elev.Dirn)
			elev.State = elevator.EBMoving
		}
		break
	}
	setCabLights(elev)
}

func OnFloorArrival(newFloor int) {
	elev.Floor = newFloor

	elevio.SetFloorIndicator(elev.Floor)

	switch elev.State {

	case elevator.EBMoving:
		if requests.ShouldStop(elev) {
			elevio.SetMotorDirection(elevio.MD_Stop)
			elevio.SetDoorOpenLamp(true)
			elev = requests.ClearAtCurrentFloor(elev)
			timer.Start(elev.Config.DoorOpenDuration)
			setCabLights(elev)
			elev.State = elevator.EBDoorOpen
		}
		break

	default:
		break
	}
}

func OnDoorTimeout() {
	switch elev.State {

	case elevator.EBDoorOpen:
		elev.Dirn = requests.ChooseDirection(elev)

		elevio.SetDoorOpenLamp(false)
		elevio.SetMotorDirection(elev.Dirn)

		if elev.Dirn == elevio.MD_Stop {
			elev.State = elevator.EBIdle
		} else {
			elev.State = elevator.EBMoving
		}
		break

	default:
		break
	}
}

func RestoreState() {
	elev, _ = elevstate.StateRestore()
}

func TransmitState(port string) {
	elevstate.StateStoreElev(port, elev)
	//fmt.Println("I'm here!")
	// Transmit json file over network
}
