package elevstate

import (
  "log"
  "os"
  "encoding/json"
  "io/ioutil"

  "../elevator"
  "../elevio"

  //"strconv"
  //"fmt"
)

type State struct{
  ID        string `json:"ID"`
  Floor     int `json:"Pos"`
  Dirn      string `json:"Dirn"`
  Requests  [4][3]bool `json:"Requests"`
}


func StateStore(e elevator.Elevator){
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
