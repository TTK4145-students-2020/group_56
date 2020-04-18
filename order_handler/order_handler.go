package order_handler

import "../elevio"
import "../fsm"
import "../elevator"
import "../timer"
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

type ActiveElevators struct {
    Elevators []string
    Connected string
    Disconnected []string
}

type Button struct {
	Floor int
	Type int //cabcall, hallcallUP eller hallcallDOWN
}


func ReceiveOrder(elevID string, slaveOrder chan<- types.Button){ //skal ta imot en ordre som sendes over nettet fra DelegateOrder
    netRecv := make(chan Order)

    for {
        select{
        case forSlave := <- netRecv:

            slaveOrder <- Button{Floor: forSlave.Floor, Type: int(forSlave.Button)}

            elevio.SetButtonLamp(forSlave.Button, forSlave.Floor, true)

        }
    }



}


func DelegateOrder(elevID string, masterOrder chan<- types.Button, slaveOrder chan<- types.Button, assignedMasterOrder <-chan Order, assignedSlaveOrder <-chan Order){ //skal sende ordren til den valgte heisen enten lokalt (til master) eller over nettet (til en slave)
    netSend := make(chan Order) // kan netSend og netReceive være i to forskjellige funksjoner?

    //TODO: legge til hvordan kanalene sendes over nettet

    for{

        select{

        case toMaster := <- assignedMasterOrder:

            masterOrder <- Button{Floor: toMaster.Floor, Type: int(toMaster.Button)}

            elevio.SetButtonLamp(toMaster.Button, toMaster.Floor, true)


        case toSlave := <- assignedSlaveOrder:
            netSend <- orderReceived


        }

    }


    /*for{

        select{

        case orderReceived := <- assignedOrder:

            if orderReceived.Button !=2 { //ikke cabcall
                //TODO: mulig det må legges til litt som gjør at dette alltid fungerer
                netSend <- orderReceived

            }

            if orderReceived == 2 || assignedOrder.AssignedTo == elevID{// er cabcall
                masterOrder <- Button{Floor: orderReceived.Floor, Type: int(orderReceived.Button)}

                elevio.SetButtonLamp(orderReceived.Button, orderReceivedFloor, true)
            }
        case orderReceived := <- netRecv:
            elevio.SetButtonLamp(orderReceived.Button, orderReceivedFloor, true)
            if orderReceived.AssignedTo == elevID {
				slaveOrder <- Button{Floor: orderReceived.Floor, Type: int(orderReceived.Button)}

        }

    }*/



}

func AssignNewOrder(masterID string, buttonPressed <- chan elevio.ButtonEvent, allStates <-chan map[string]elevator.Elevator, elevatorUpdate <-chan ActiveElevators, assignedMasterOrder chan Order, assignedSlaveOrder chan Order )
// skal registrere en ny ordre og finne den beste heisen for denne ordren

var activeElevators []string
var states map[string]elevator.Elevator


    for{
        select {

        case updatedState := <- allStates:
            states = updatedState


        case newElevator := <- elevatorUpdate:
            activeElevators = newElevator.Elevators


        case btn := <- buttonPressed:

            currentStates := make(map[string]elevator.Elevator)

            for _, id:= range peers {
                if state, ok:=states[id]; ok {
                    currentStates[id] = state
                }
            }

        bestElev := findBestElev(btn, elevID, currentStates )

        newOrder := Order{btn.Floor, btn.Button, bestElev}

        if bestElev == masterID {
            assignedMasterOrder <- newOrder
        }
        if bestElev != masterID {
            assignedSlaveOrder <- newOrder
        }


        }

    }
}

}




func findBestElev(hallBtn elevio.ButtonEvent ,elevID string, states map[string]elevator.Elevator  ){ //finner den beste heisen ut i fra kostfunksjonen

	lowestCost := math.MaxInt64
	bestElev := elevID

	for id, state := range states {//går gjennom alle states til hver elevID
        state.Requests[btn.Floor][btn.Button] = 2
        c = elevResponseTime(id)
		if c < lowestCost{
			lowestCost = c
			bestElev = id
		}

	}

	return bestElev //string


}

func elevResponseTime(e elevator.Elevator) int {//finner tiden det tar før heisen er "fri" (idle) til gjennomføre en ny request
    const TRAVEL_TIME := 2000
    const DOOR_OPEN_TIME := timer.Start(e.config.DoorOpenDuration)
    D_Stop := elevio.MD_Stop

    duration := 0;

    switch e.behaviour {
    case elevator.EB_Idle:
        e.dirn = requests_chooseDirection(e);
        if(e.dirn == D_Stop){
            return duration; //returnerer fordi heisen idle = "fri"
        }
        break;

    case elevator.EB_Moving:
        duration += TRAVEL_TIME/2;
        e.floor += e.dirn;
        break;

    case elevator.EB_DoorOpen:
        duration -= DOOR_OPEN_TIME/2;
    }


    while(true){
        if(requests_shouldStop(e)){
            e = requests_clearAtCurrentFloor(e, NULL);
            duration += DOOR_OPEN_TIME;
            e.dirn = requests_chooseDirection(e);
            if(e.dirn == D_Stop){
                return duration; //returnerer dette når den finner ut at det ikke er flere steder å gå
            }
        }
        e.floor += e.direction;
        duration += TRAVEL_TIME;
    }
}
