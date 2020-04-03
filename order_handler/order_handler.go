package over_handler

import "../elevio"
import "../fsm"
import "../elevator"
import "../timer"
import "math"


func delegateOrder(){
	

}



func findBestElev(elevID string,  ){

	lowestCost := 
	bestElev := elevID 

	for id ?? {
		c = timeToIdle(id)
		if c < lowestCost{
			lowestCost = c
			bestElev = id
		}

	}

	return bestElev //string




}

func elevResponseTime(e elevator.Elevator){//finner tiden det tar før heisen er "fri" (idle) til gjennomføre en ny request
    TRAVEL_TIME := 2000
    DOOR_OPEN_TIME := timer.Start(e.config.DoorOpenDuration)
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

