package order_handler

import (
    "math"

    "../elevio"
    "../requests"
    "../fsm"
    "../elevator"
    "../timer"
    "../elevstate"
)

// se på SystemState.json, cleare gamle unassignedRequests, extract nye ordre
func HandleSystemStateFromMaster(systemState elevstate.SystemState) (newOrders []elevio.ButtonEvent, hallLights [4][2]bool, err error) {
  localState, err := elevstate.RetrieveState()
  
  if err != nil {
    log.Println(err)
    return
  }
  
  // Finner master sin versjon av state
  var remoteState elevstate.State
  for i, state := range systemState.states {
    if state.ID == localState.ID {
      remoteState = state
      break
    }
  }
  
  // Clearer de unassignedRequestene (fra localState) som master har sett
  for _, r1 := range remoteState.unassignedRequests {
    for i, r2 := range localState.unassignedRequests {
      if r1.Floor == r2.Floor && r1.Button == r2.Button {
        // Slette ett element på index i
        copy(localState.unassignedRequests[i:], localState.unassignedRequests[i+1:])
        localState.unassignedRequests[len(localState.unassignedRequests) - 1] = {}
        localState.unassignedRequests = localState.unassignedRequests[:len(localState.unassignedRequests)-1]
        
        break
      }
    }
  }
  
  // Extract nye ordre og hallLights
  newOrders = remoteState.newOrders
  hallLights = systemState.hallLights
  
  //remoteState.newOrders = []
  //localState.NewOrders = append(localState.NewOrders, newOrders...)
  localState.NewOrders = newOrders
  
  // Lagre State
  err = elevstate.StateStore(localState)
  // Tenker da ikke å sende state tilbake til master med en gang, etter som det vil gjøres når chaffeur tar imot de nye ordrene
  if err != nil {
      fmt.Println(err)
      return
  }
}

func updateHallLights(systemState elevstate.SystemState) (hallLights [4][2]bool) {
    for f := 0; f < elevator.NumFloors; f++ {
        btnTypes := []elevio.ButtonType{elevio.BT_HallUp, elevio.BT_HallDown}
        for bt := range btnTypes {
            for state := range systemState.states {
                if state.Requests[f][bt] {
                    hallLights[f][bt] = true
                    break
                }
                hallLights[f][bt] = false
            }
        }
    }
}

// ta inn staten til slaven som string (json), cleare newOrder i SystemState.json, se etter unassigned hos alle slaver
func HandleStateFromSlave(slaveState elevstate.State) (unassignedRequests []elevio.ButtonEvent, hallLights [4][2]bool, err error) {
  systemState, err := elevstate.RetrieveSystemState()
  
  if err != nil {
    log.Println(err)
    return
  }
  
  // Finner index i system
  var systemIndex int
  for i, state := range systemState.states {
    if state.ID == slaveState.ID {
      systemIndex = i
      break
    }
  }
  
  // Clearer de newOrder-ene (fra system) som slave har sett
  for _, r1 := range slaveState.NewOrders {
    for i, r2 := range  systemState.states[i].NewOrders {
      if r1.Floor == r2.Floor && r1.Button == r2.Button {
        // Slette ett element på index i
        copy(systemState.states[systemIndex].NewOrders[i:], systemState.states[systemIndex].NewOrders[i+1:])
        systemState.states[systemIndex].NewOrders[len(systemState.states[systemIndex].NewOrders) - 1] = {}
        systemState.states[systemIndex].NewOrders = systemState.states[systemIndex].NewOrders[:len(systemState.states[systemIndex].NewOrders)-1]
        
        break
      }
    }
  }
  
  unassignedRequests = slaveState.unassignedRequests

  hallLights = updateHallLights(systemState)
  systemState.HallLights = hallLights
  
  systemState.states[systemIndex].unassignedRequests = unassignedRequests
  systemState.states[systemIndex].Floor = slaveState.Floor
  systemState.states[systemIndex].Dirn = slaveState.Dirn
  systemState.states[systemIndex].Requests = slaveState.Requests

  err = elevstate.SystemStore(systemState)
  
  if err != nil {
    log.Println(err)
  }
}

// lagre ny request i unassignedRequest
func NewRequest(request elevio.ButtonEvent) (err error) {
    localState, err := elevstate.RetrieveState()

    if err != nil {
        fmt.Println(err)
        return
    }

    localState.unassignedRequest = append(localState.unassignedRequest, request)

    err = elevstate.StateStore(localState)

    if err != nil {
        log.Println(err)
        return
    }
}

func AssignNewOrder(masterID string, newRequest elevio.ButtonEvent) (toMaster bool, hallLights [4][2]bool, err error) {
  
  systemState, err := elevstate.RetrieveSystemState()

  if err != nil [
      log.Println(err)
  ]
  
  bestIndex := findBestElevIndex(newRequest, systemState.states)
  
  systemState.states[bestIndex].newOrders = append(systemState.states[bestIndex].newOrders, newRequest)
  
  systemState.HallLights[newRequest.Floor][newRequest.Button] = true

  toMaster = systemState.states[bestIndex].ID == masterID
  hallLights = systemState.HallLights
  
  err = elevstate.SystemStore(systemState)

  if err != nil {
      log.Println(err)
  }
}

func findBestElevIndex(hallBtn elevio.ButtonEvent , states []elevstate.State  ) int { 
  bestTime := math.MaxInt64
  bestIndex = 0
  for i, state := range states {
    e := elevator.Uninitialized()
    e.Floor = state.Floor
    e.Dirn = state.Dirn
    e.Requests = state.Requests
    e.Requests[btn.Floor][btn.Button] = true
    
    tti := timeToIdle(e)
    if tti < bestTime {
      bestTime = tti
      bestIndex = i
    }
  }

	return i //int
}

func timeToIdle(e elevator.Elevator) int {
    const TRAVEL_TIME := 2000
    const DOOR_OPEN_TIME := timer.Start(e.config.DoorOpenDuration)
    D_Stop := elevio.MD_Stop

    duration := 0;
    
    switch e.State {
    case elevator.EB_Idle:
        e.Dirn = requests.ChooseDirection(e);
        if e.dirn == D_Stop {
            return duration;
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
            e.Dirn = requests.chooseDirection(e);
            if e.Dirn == D_Stop {
                return duration; 
            }
        }
        e.Floor += e.Direction;
        duration += TRAVEL_TIME;
    }
}