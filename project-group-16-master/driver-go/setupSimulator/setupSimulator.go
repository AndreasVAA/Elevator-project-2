package setupSimulator

// This module implements functions for faster setup of simulators
import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"driver-go/elevio"
)

func SetupOneSimulator(initElevSpec []string) {
	var id, err = strconv.Atoi(initElevSpec[1])
	if err != nil {
		panic(1)
	}
	if id < 9999 {
		terminal := exec.Command("gnome-terminal", "-x", "sh", "-c", "go run main.go '"+initElevSpec[2]+"' '"+initElevSpec[1]+"'")
		terminal.Run()
		time.Sleep(1 / 3 * time.Second)
	} else {
		fmt.Println("Setter opp simulator! ")
		terminal := exec.Command("gnome-terminal", "-x", "sh", "-c", "./SimElevatorServer --port "+initElevSpec[1])
		terminal.Run()
		time.Sleep(1 * time.Second)
		elevio.Init("localhost:"+initElevSpec[1], elevio.N_FLOORS)
	}
}
