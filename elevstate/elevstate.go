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

type System struct{
    states []State `json:"States"`
}

type State struct{
  ID        string `json:"ID"`
  Floor     int `json:"Pos"`
  Dirn      string `json:"Dirn"`
  Requests  [4][3]bool `json:"Requests"`
}

var mux sync.Mutex

const statepath = "./elevstate/state.json"
const syspath = "./elevstate/systemState.json"

// Must be called from main, updates or generates state
func StateInit(port string) (error) {
  var requests [4][3]bool
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
func StateStoreElev(port string, e elevator.Elevator) (error){
  state := State{
    ID: port,
    Floor: e.Floor,
    Dirn: dirToString(e.Dirn),
    Requests: e.Requests,
  }

  var jsonData []byte
  jsonData, err := json.Marshal(state)
  if err != nil {
      return err
  }

  mux.Lock()
  err = ioutil.WriteFile(statepath, jsonData, 0644)
  mux.Unlock()
  return err
}

// stores state (State struct as byte-array) in file state.Json
func StateStore(state []byte) (err error){
  mux.Lock()
  err = ioutil.WriteFile(statepath, state, 0644)
  mux.Unlock()
  return
}

// stores system (System struct as byte-array) in file system.json
func SystemStore(system []byte) (err error){
  mux.Lock()
  err = ioutil.WriteFile(syspath, system, 0644)
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
      return genSystemFile(System{[]State{state}})
  }

  systembytes, err := ioutil.ReadAll(jsonFile)
  if err != nil {
      jsonFile.Close()
      mux.Unlock()
      return err
  }
  jsonFile.Close()
  mux.Unlock()

  // err = json.Unmarshal(systembytes, &system)
  system = unmarshalSystem(systembytes)
  if err != nil {
      return err
  }

  existance := false
  for i := 0; i < len(system.states); i++ {
    if(system.states[i].ID == state.ID){
      system.states[i] = state
      existance = true
    }
  }
  if(!existance){
    system.states = append(system.states, state)
  }

  // systembytes, err = json.Marshal(system)
  systembytes = marshalSystem(system)
  if err != nil {
      return err
  }

  err = SystemStore(systembytes)
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
  if err != nil {
    jsonFile.Close()
    mux.Unlock()
    return e, err
  }
  jsonFile.Close()
  mux.Unlock()

  var state State
  err = json.Unmarshal(statebytes, &state)
  if err != nil {
      return e, err
  }

  e.Floor = state.Floor
  e.Dirn = stringToDir(state.Dirn)
  e.Requests = state.Requests
  e.State = elevator.EBIdle
  e.Config.ClearRequestVariant = elevator.CVALL
	e.Config.DoorOpenDuration = 3
  return e, nil
}

func dirToString(dirn elevio.MotorDirection) (dirnS string){
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

func stringToDir(dirnS string) (dirn elevio.MotorDirection){
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

  // var system System
  // err = json.Unmarshal(systembytes, &system)
  system := unmarshalSystem(systembytes)
  // if err != nil {
  //     return nil, err
  // }
  for i := 0; i < len(system.states); i++ {
    if(system.states[i].ID == port){
      statebytes, err := json.Marshal(system.states[i])
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
    var systembytes []byte
    systembytes = marshalSystem(system)
    err = SystemStore(systembytes)
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
    statebytes, err := json.Marshal(oldstate)
    if err != nil {
      return err
    }

    err = StateStore(statebytes)
    return err

  }else{
    var statebytes []byte
    statebytes, err = json.Marshal(state)
    if err != nil{
      return err
    }
    err = StateStore(statebytes)
    return err
  }

  return nil
}

func marshalSystem(system System) ([]byte) {
  var systemBytes []byte
  for i := 0; i < len(system.states); i++ {
    statebytes, err := json.Marshal(system.states[i])
    if err == nil {
      systemBytes = append(systemBytes, statebytes...)
      systemBytes = append(systemBytes, []byte("||")...)
    }
  }
  return systemBytes
}
func unmarshalSystem(systemBytes []byte) (System){
  var system System
  allstates := bytes.Split(systemBytes, []byte("||"))

  for i := 0; i < len(allstates); i++ {
    var state State
    err := json.Unmarshal(allstates[i], &state)
    if err == nil {
      system.states = append(system.states, state)
    }
  }
  return system
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
