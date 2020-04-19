package order_handler

import "../elevio"
import "../requests"
import "../fsm"
import "../elevator"
import "../timer"
import "/..elevstate"
import "../peers"
import "math"

type ElevQueue struct {
	QueueSystem [4][4]int
	CabCall     [4]int
	HallCall    [4][2]int
	ID          string
}

type Order struct {
    Floor int
    Button elevio.ButtonType
    AssignedTo string //hvilken heis som har orderen
}



func DelegateOrder(elevID string, btn chan<- elevio.ButtonEvent,  assignedOrder <-chan Order){ //skal sende ordren til den valgte heisen enten lokalt (til master) eller over nettet (til en slave)
    netSend := make(chan Order) 
    netRecv := make(chan Order)

    //TODO: legge til hvordan kanalene sendes over nettet



    for{

        select{
        
        case orderReceived := <- assignedOrder:

            if btn! =2 { //ikke cabcall
                //TODO: mulig det må legges til litt som gjør at dette alltid fungerer
                netSend <- orderReceived

            }

            if orderReceived.AssignedTo == elevID || orderReceived.Button == 2 {// er cabcall
                btn <- elevio.ButtonEvent{Floor: orderReceived.Floor, Button: int(orderReceived.Button)}//litt skeptisk til denne

                elevio.SetButtonLamp(orderReceived.Button, orderReceivedFloor, true)
            }

        case orderReceived := <- netRecv:
            elevio.SetButtonLamp(orderReceived.Button, orderReceivedFloor, true)

            if orderReceived.AssignedTo == elevID {
				btn <- elevio.ButtonEvent{Floor: orderReceived.Floor, Button: int(orderReceived.Button)} //litt skeptisk til denne

        }

    }*/

	

}

func AssignNewOrder(elevID string, buttonPressed <- chan elevio.ButtonEvent, allStates <-chan map[string]elevator.Elevator, peerUpdate <-chan peers.PeerUpdate, assignedOrder chan Order) 
// skal registrere en ny ordre og finne den beste heisen for denne ordren

var activeElevators []string
var states map[string]elevator.Elevator


    for{
        select {

        case updatedStates := <- allStates:
            states = updatedStates


        case newElevator := <- elevatorUpdate:
            activeElevators = newElevator.Peers

        
        case btn := <- buttonPressed:

            currentStates := make(map[string]elevator.Elevator)

            for _, id:= range activeElevators {
                if state, ok:=states[id]; ok {
                    currentStates[id] = state
                }
            }
        
            bestElev := findBestElev(btn, elevID, currentStates )

            newOrder := Order{btn.Floor, btn.Button, bestElev}

            assignedOrder <- newOrder

        }

    }
}






func findBestElev(hallBtn elevio.ButtonEvent , elevID string, states map[string]elevator.Elevator  ){ //finner den beste heisen ut i fra kostfunksjonen

	bestTime := math.MaxInt64
	bestElev := elevID 

	for id, state := range states {//går gjennom alle states til hver elevID
        state.Requests[btn.Floor][btn.Button] = 2
		if timeToIdle(id) < bestTime{
			lowestCost = timeToIdle(id)
			bestElev = id
		}

	}

	return bestElev //string


}

func timeToIdle(e elevator.Elevator) int {//finner tiden det tar før heisen er "fri" (idle) til gjennomføre en ny request
    const TRAVEL_TIME := 2000
    const DOOR_OPEN_TIME := timer.Start(e.config.DoorOpenDuration)
    D_Stop := elevio.MD_Stop

    duration := 0;
    
    switch e.State {
    case elevator.EB_Idle:
        e.Dirn = requests.ChooseDirection(e);
        if e.dirn == D_Stop {
            return duration; //returnerer fordi heisen idle = "fri" 
        }
        break;

    case elevator.EB_Moving:
        duration += TRAVEL_TIME/2;
        e.Floor += e.Dirn;
        break;

    case elevator.EB_DoorOpen:
        duration -= DOOR_OPEN_TIME/2;
    }


    for {
        if requests.ShouldStop(e){
            e = requests.ClearAtCurrentFloor(e, nil); // TODO: må gjøre om ClearAtCurrentFloor i  requests.go
            duration += DOOR_OPEN_TIME;
            e.Dirn = requests_chooseDirection(e);
            if e.Dirn == D_Stop {
                return duration; //returnerer dette når den finner ut at det ikke er flere steder å gå
            }
        }
        e.Floor += e.Direction;
        duration += TRAVEL_TIME;
    }
}

func jsonToStateMap(elevID string) (states map[string]elevator.Elevator){

    e := elevstate.StateRestore()

    m = make(map[string]elevator.Elevator)

    m[elevID] = e

    return e

  }

