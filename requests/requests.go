package requests

import (
	"../elevator"
	"../elevio"
)

func Above(e elevator.Elevator) (req bool) {
	for f := e.Floor + 1; f < elevator.NumFloors; f++ {
		for btn := 0; btn < elevator.NumButtons; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func Below(e elevator.Elevator) (req bool) {
	for f := 0; f < e.Floor; f++ {
		for btn := 0; btn < elevator.NumButtons; btn++ {
			if e.Requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func ChooseDirection(e elevator.Elevator) (dirn elevio.MotorDirection) {
	switch e.Dirn {
	case elevio.MD_Up:
		if Above(e) {
			return elevio.MD_Up
		} else if Below(e) {
			return elevio.MD_Down
		} else {
			return elevio.MD_Stop
		}
	case elevio.MD_Down:
		if Below(e) {
			return elevio.MD_Down

		} else if Above(e) {
			return elevio.MD_Up
		} else {
			return elevio.MD_Stop
		}
	case elevio.MD_Stop:
		if Below(e) {
			return elevio.MD_Down
		} else if Above(e) {
			return elevio.MD_Up
		} else {
			return elevio.MD_Stop
		}
	default:
		return elevio.MD_Stop
	}
	return
}

func ShouldStop(e elevator.Elevator) (stop bool) {
	switch e.Dirn {
	case elevio.MD_Down:
		stop = (e.Requests[e.Floor][elevio.BT_HallDown] || e.Requests[e.Floor][elevio.BT_Cab] || !Below(e))
		return
	case elevio.MD_Up:
		stop = (e.Requests[e.Floor][elevio.BT_HallUp] || e.Requests[e.Floor][elevio.BT_Cab] || !Above(e))
		return
	case elevio.MD_Stop:
		return true
	default:
		return true
	}
}

func ClearAtCurrentFloor(e elevator.Elevator) elevator.Elevator {
	switch e.Config.ClearRequestVariant {

	case elevator.CVALL:
		for btn := 0; btn < elevator.NumButtons; btn++ {
			e.Requests[e.Floor][btn] = false
		}
		break

	case elevator.CVInDirn:
		e.Requests[e.Floor][elevio.BT_Cab] = false
		switch e.Dirn {

		case elevio.MD_Up:
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			if !Above(e) {
				e.Requests[e.Floor][elevio.BT_HallDown] = false
			}
			break

		case elevio.MD_Down:
			e.Requests[e.Floor][elevio.BT_HallDown] = false
			if !Below(e) {
				e.Requests[e.Floor][elevio.BT_HallUp] = false
			}
			break

		case elevio.MD_Stop:
		default:
			e.Requests[e.Floor][elevio.BT_HallUp] = false
			e.Requests[e.Floor][elevio.BT_HallDown] = false
			break
		}
		break
	default:
		break
	}
	return e
}
