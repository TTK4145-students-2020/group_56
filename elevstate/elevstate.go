package elevstate

import (
  "log"
  "os"
  "encoding/json"
  "io/ioutil"

  "../elevator"
  "../elevio"

  //"strconv"
  "fmt"
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

func StateStoreElev(e elevator.Elevator){
  state := State{
    ID: "xxx.xxx.xxx.xxx",//?
    Floor: e.Floor,
    Dirn: dirToString(e.Dirn),
    Requests: e.Requests,
  }

  var jsonData []byte
  jsonData, err := json.Marshal(state)
  if err != nil {
      log.Println(err)
  }
  err = ioutil.WriteFile("state.json", jsonData, 0644)
  if err != nil {
      log.Println(err)
  }
}

func StateStore(state []byte) (err error){
  err = ioutil.WriteFile("state.json", state, 0644)
  return
}

func SystemStore(system []byte) (err error){
  err = ioutil.WriteFile("systemState.json", system, 0644)
  return
}

func SystemStateUpdate(statebytes []byte) (err error){
  var state State
  var system System

  fmt.Println("før unmarshal")
  err = json.Unmarshal(statebytes, &state)
  if err!= nil {
    return
  }
  fmt.Println("etter unmarshal")

  jsonFile, err := os.Open("systemState.json")
  if err != nil {
      //return
      system = System{
        states: []State{state},
      }
      //system.states = append(system.states, state)
      var systembytes []byte
      systembytes, err = json.Marshal(system)
      if err != nil{
        return
      }

      err = SystemStore(systembytes)
      return

  }

  systembytes, err := ioutil.ReadAll(jsonFile)
  if err != nil {
      return
  }
  jsonFile.Close()

  err = json.Unmarshal(systembytes, &system)
  if err != nil {
      return
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

  systembytes, err = json.Marshal(system)
  if err != nil {
      return
  }

  err = SystemStore(systembytes)
  return
}

func StateRestore() (e elevator.Elevator){
  jsonFile, err := os.Open("state.json")
  if err != nil {
      log.Println(err)
  }
  defer jsonFile.Close()

  var state State
  statebytes, err := ioutil.ReadAll(jsonFile)
  if err != nil {
      log.Println(err)
  }

  err = json.Unmarshal(statebytes, &state)
  if err != nil {
      log.Println(err)
  }

  e.Floor = state.Floor
  e.Dirn = stringToDir(state.Dirn)
  e.Requests = state.Requests
  e.State = elevator.EBIdle
  e.Config.ClearRequestVariant = elevator.CVALL
	e.Config.DoorOpenDuration = 3
  return
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

func RetrieveStateBytes() (statebytes []byte, err error){
  jsonFile, err := os.Open("state.json")
  if err != nil {
      //log.Println(err)
      statebytes = nil
      return
  }
  defer jsonFile.Close()

  statebytes, err = ioutil.ReadAll(jsonFile)
  if err != nil {
      //log.Println(err)
      statebytes = nil
      return
  }

  return
}

func RetrieveSystemStateBytes() (statebytes []byte, err error){
  jsonFile, err := os.Open("systemState.json")
  if err != nil {
      //log.Println(err)
      statebytes = nil
      return
  }
  defer jsonFile.Close()

  statebytes, err = ioutil.ReadAll(jsonFile)
  if err != nil {
      //log.Println(err)
      statebytes = nil
      return
  }

  return
}

//returns the statebytes of the elevator connected on port port, nil if error occurs or no elevator connected on port.
func RetrieveRemoteStateBytes(port string) (statebytes []byte, err error){
  jsonFile, err := os.Open("systemState.json")
  if err != nil {
      statebytes = nil
      return
  }
  defer jsonFile.Close()

  systembytes, err := ioutil.ReadAll(jsonFile)
  if err != nil {
      statebytes = nil
      return
  }
  var system System
  err = json.Unmarshal(systembytes, &system)
  if err != nil {
      statebytes = nil
      return
  }
  for i := 0; i < len(system.states); i++ {
    if(system.states[i].ID == port){
      statebytes, err = json.Marshal(system.states[i])
      return
    }
  }
  statebytes = nil
  return
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
