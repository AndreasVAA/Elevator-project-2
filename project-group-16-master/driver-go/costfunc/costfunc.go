package costfunc

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	"driver-go/elevio"
	"driver-go/orders"
)

type HRAElevState struct {
	Behavior    string                `json:"behaviour"`
	Floor       int                   `json:"floor"`
	Direction   string                `json:"direction"`
	CabRequests [elevio.N_FLOORS]bool `json:"cabRequests"`
}

type ElevatorInfoToBroadcast struct {
	ElevState HRAElevState
	Id        string
	SyncHall  [elevio.N_BUTTONS - 1][elevio.N_FLOORS]orders.OrderStatus
}

type HRAInput struct {
	HallRequests [elevio.N_FLOORS][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

func RunCostFunc(input HRAInput) {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		fmt.Println("json.Marshal error: ", err)
		return
	}

	ret, err := exec.Command("./costfunc/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		fmt.Println("exec.Command error: ", err)
		fmt.Println(string(ret))
		return
	}

	output := new(map[string][][2]bool)
	err = json.Unmarshal(ret, &output)
	if err != nil {
		fmt.Println("json.Unmarshal error: ", err)
		return
	}

	for key, value := range *output {
		if elevio.LocalId == key {
			for button := 0; button < elevio.N_BUTTONS-1; button++ {
				for floor := 0; floor < elevio.N_FLOORS; floor++ {
					orders.SetLocalOrder(floor, elevio.ButtonType(button), false)
					if button == int(elevio.BT_HallUp) && value[floor][button] == true {
						orders.SetLocalOrder(floor, elevio.BT_HallUp, true)
					} else if button == int(elevio.BT_HallDown) && value[floor][button] == true {
						orders.SetLocalOrder(floor, elevio.BT_HallDown, true)
					}
				}
			}
		}
	}
}
