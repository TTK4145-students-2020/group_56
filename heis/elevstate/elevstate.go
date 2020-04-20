package elevstate

//  TODO: legge til funksjonalitet for å oppdatere (dersom nødvendig) posisjon ved omstart
//  TODO: legge til funksjon for initialisering av systemstate hos master

import (
  "os"
  "encoding/json"
  "io/ioutil"
  "bytes"

  "../elevator"
  "../elevio"

  "sync"
  // "fmt"
)
type Lights struct{
  HallLights [4][2]bool `json:"Lights"`
}


type System struct{
  HallLights [4][2]bool
  States []State `json:"States"`
}

// legg til neworders og unassignedrequests (type: elevio.buttonevent)

type State struct{
  ID          string `json:"ID"`
  Floor       int `json:"Pos"`
  Dirn        string `json:"Dirn"`
  Requests    [4][3]bool `json:"Requests"`

  NewOrders   []elevio.ButtonEvent `json:"NewOrders"`
  NewRequests []elevio.ButtonEvent `json:"NewRequests"`

  Active      bool `json:Active`
}

var mux sync.Mutex

const statepath = "./elevstate/state.json"
const syspath = "./elevstate/systemState.json"

// Must be called from main, updates or generates state
func StateInit(port string) (error) {
  var requests [4][3]bool
  var newOrders []elevio.ButtonEvent
  var newRequests []elevio.ButtonEvent

  for i := 0; i < 4; i++ {
    for j := 0; j < 3; j++ {
      requests[i][j] = false
    }
  }

  state := State{
    ID: port,
    Floor: -1,
    Dirn: "",
    Requests: requests,
    NewOrders: newOrders,
    NewRequests: newRequests,
    Active: true,
  }

  err := genStateFile(state)
  return err
}

func SystemInit() (error) {
  statebytes, err := RetrieveStateBytes()
  if err != nil {
    return err
  }

  return SystemStateUpdate(statebytes)
}

// stores e as a state.Json file
func StateStoreElev(e elevator.Elevator, newRequests []elevio.ButtonEvent) (error){
  state, err := RetrieveState()
  if err != nil {
    return err
  }

  state.Floor = e.Floor
  state.Dirn = DirToString(e.Dirn)
  state.Requests = e.Requests

  var existance bool
  for _, request := range newRequests {
    existance = false
    for _, old := range state.NewRequests {
      if request == old {
        existance = true
        continue
      }
    }
    if (!existance) {
      state.NewRequests = append(state.NewRequests, request)
    }
  }

  var jsonData []byte
  jsonData, err = json.Marshal(state)
  if err != nil {
      return err
  }

  mux.Lock()
  err = ioutil.WriteFile(statepath, jsonData, 0644)
  mux.Unlock()
  return err
}

// stores state in file state.Json
func StateStore(state State) (error){

  statebytes, err := json.Marshal(state)
  if err != nil{
    return err
  }
  mux.Lock()
  err = ioutil.WriteFile(statepath, statebytes, 0644)
  mux.Unlock()
  return err
}

// stores system in file system.json
func SystemStore(system System) (err error){

  systembytes := marshalSystem(system)

  mux.Lock()
  err = ioutil.WriteFile(syspath, systembytes, 0644)
  mux.Unlock()
  return
}

// Adds/updates single elevator (statebytes) in systemstate.json
func SystemStateUpdate(statebytes []byte) (error){
  var state State
  var system System

  err := json.Unmarshal(statebytes, &state)
  if err!= nil {
    return err
  }

  mux.Lock()
  jsonFile, err := os.Open(syspath)
  if err != nil {
      mux.Unlock()
      var hallLights [4][2]bool
      return genSystemFile(System{hallLights, []State{state}})
  }

  systembytes, err := ioutil.ReadAll(jsonFile)
  if err != nil {
      jsonFile.Close()
      mux.Unlock()
      return err
  }
  jsonFile.Close()
  mux.Unlock()

  system = UnmarshalSystem(systembytes)
  if err != nil {
      return err
  }

  existance := false
  for i := 0; i < len(system.States); i++ {
    if(system.States[i].ID == state.ID){
      system.States[i] = state
      existance = true
    }
  }
  if(!existance){
    system.States = append(system.States, state)
  }

  err = SystemStore(system)
  return err
}

// Reads state.json and returns as Elevator struct
func StateRestore() (elevator.Elevator, error) {
  var e elevator.Elevator

  mux.Lock()
  jsonFile, err := os.Open(statepath)
  if err != nil {
    mux.Unlock()
    return e, err
  }

  statebytes, err := ioutil.ReadAll(jsonFile)
  jsonFile.Close()
  mux.Unlock()
  if err != nil {
    return e, err
  }

  var state State
  err = json.Unmarshal(statebytes, &state)
  if err != nil {
      return e, err
  }

  e.Floor = state.Floor
  e.Dirn = StringToDir(state.Dirn)
  e.Requests = state.Requests
  e.State = elevator.EBIdle
  e.Config.ClearRequestVariant = elevator.CVALL
	e.Config.DoorOpenDuration = 3
  return e, nil
}

func DirToString(dirn elevio.MotorDirection) (dirnS string){
  switch dirn {
  case elevio.MD_Up:
    dirnS = "MD_Up"
    break
  case elevio.MD_Down:
    dirnS = "MD_Down"
    break
  default:
    dirnS = "MD_Stop"
    break
  }
  return
}

func StringToDir(dirnS string) (dirn elevio.MotorDirection){
  switch dirnS {
  case "MD_Up":
    dirn = elevio.MD_Up
    break
  case "MD_Down":
    dirn = elevio.MD_Down
    break
  default:
    dirn = elevio.MD_Stop
    break
  }
  return
}

// Reads state.json and returns as State struct (in byte-array form)
func RetrieveStateBytes() ([]byte, error){
  mux.Lock()
  jsonFile, err := os.Open(statepath)
  if err != nil {
      mux.Unlock()
      return nil, err
  }

  statebytes, err := ioutil.ReadAll(jsonFile)
  jsonFile.Close()
  mux.Unlock()
  if err != nil {
      return nil, err
  }

  return statebytes, nil
}

// Reads system.json and returns as System struct (in byte-array form)
func RetrieveSystemStateBytes() ([]byte, error){
  mux.Lock()
  jsonFile, err := os.Open(syspath)
  if err != nil {
      mux.Unlock()
      return nil, err
  }

  statebytes, err := ioutil.ReadAll(jsonFile)
  jsonFile.Close()
  mux.Unlock()
  if err != nil {
      return nil, err
  }

  return statebytes, nil
}

//returns the statebytes of the elevator connected on port port, nil if error occurs or no elevator connected on port.
func RetrieveRemoteStateBytes(port string) ([]byte, error){
  mux.Lock()
  jsonFile, err := os.Open(syspath)
  if err != nil {
      mux.Unlock()
      return nil, err
  }

  systembytes, err := ioutil.ReadAll(jsonFile)
  jsonFile.Close()
  mux.Unlock()
  if err != nil {
      return nil, err
  }

  system := UnmarshalSystem(systembytes)

  for i := 0; i < len(system.States); i++ {
    if(system.States[i].ID == port){
      statebytes, err := json.Marshal(system.States[i])
      return statebytes, err
    }
  }
  return nil, nil
}

func genSystemFile(system System) (error){
  mux.Lock()
  file, err := os.Open(syspath)
  if err != nil {
    mux.Unlock()
    err = SystemStore(system)
    if err != nil {
      return err
    }
    return nil
  }

  file.Close()
  mux.Unlock()
  return nil
}

func genStateFile(state State) (error){
  existance  := true
  var statebytes []byte
  var err error

  mux.Lock()
  jsonFile, err := os.Open(statepath)
  if err != nil {
    mux.Unlock()
    existance = false
  }else{
    statebytes, err = ioutil.ReadAll(jsonFile)
    jsonFile.Close()
    mux.Unlock()
    if err != nil {
      existance = false
    }
  }

  if existance {
    var oldstate State
    err = json.Unmarshal(statebytes, &oldstate)
    if err != nil {
      return err
    }
    oldstate.ID = state.ID

    err = StateStore(oldstate)
    return err

  }else{
    err = StateStore(state)
    return err
  }

  return nil
}

func marshalSystem(system System) ([]byte) {
  var systemBytes []byte

  systemBytes, _ = json.Marshal(Lights{system.HallLights})
  systemBytes = append(systemBytes, []byte("||")...)

  for i := 0; i < len(system.States); i++ {
    statebytes, err := json.Marshal(system.States[i])
    if err == nil {
      systemBytes = append(systemBytes, statebytes...)
      systemBytes = append(systemBytes, []byte("||")...)
    }
  }
  return systemBytes
}

func UnmarshalSystem(systemBytes []byte) (System){
  var system System
  allstates := bytes.Split(systemBytes, []byte("||"))

  var lights Lights
  json.Unmarshal(allstates[0], &lights)


  system.HallLights = lights.HallLights

  for i := 1; i < len(allstates); i++ {
    var state State
    err := json.Unmarshal(allstates[i], &state)
    if err == nil {
      system.States = append(system.States, state)
    }
  }
  return system
}

func RetrieveSystemState() (System, error){
  systembytes, err := RetrieveSystemStateBytes()
  system := UnmarshalSystem(systembytes)
  return system, err

}

func RetrieveState() (State, error){
  var state State
  statebytes, err := RetrieveStateBytes()
  err = json.Unmarshal(statebytes, &state)
  return state, err

}

func StateFromBytes(statebytes []byte) (State, error){
  var state State
  err := json.Unmarshal(statebytes, &state)
  return state, err
}

/*
func Test(){
  teststate := State{
    ID: "255.255.255.255",
    Floor: 3,
    Requests: [4][3]bool{{false, false, true}, {false, true, false}, {true, false, false}, {true, false, true}},
  }

  var jsonData []byte
  jsonData, err := json.Marshal(teststate)
  ioutil.WriteFile("test.json", jsonData, 0644)
  if err != nil {
      log.Println(err)
  }

  fmt.Println(string(jsonData))

  jsonFile, err := os.Open("test.json")
  if err != nil {
       fmt.Println(err)
  }

  fmt.Println("Successfully Opened state.json")

  defer jsonFile.Close()

  byteValue, _ := ioutil.ReadAll(jsonFile)

  var state State

  json.Unmarshal(byteValue, &state)

  fmt.Println("Elevator ID: ", state.ID)
  fmt.Println("Floor: ", strconv.Itoa(state.Floor))
  fmt.Println("Requests: ")
  for i := 0; i<4; i++{
    for j := 0; j<3; j++{
      fmt.Print(state.Requests[i][j], " ")
    }
    fmt.Println("")
  }
}
*/
