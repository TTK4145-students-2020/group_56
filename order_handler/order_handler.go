package order_handler

import "../elevio"
import "../fsm"
import "../elevator"
import "../timer"
import "math"


func delegateNewOrder(elevID string, buttonPressed <-chan elevio.ButtonEvent){ //skal legge til ordren i køen til den valgte heisen



	

}

func assignNewOrder() // skal ta registrere en ny ordre og finne den beste heisen 

for{
    select {

    case btn := buttonPressed:

        bestElev := findBestElev(btn, , elevID )

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

