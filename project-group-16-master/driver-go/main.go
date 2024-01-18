
package main

import (
	"driver-go/communication"
	"driver-go/costfunc"
	"driver-go/elevio"
	"driver-go/finiteStateMachine"
	"driver-go/network/bcast"
	"driver-go/network/peers"
	"driver-go/orders"
	"fmt"
	"os"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	initElevSpec := os.Args
	//USE FOR PHYSICAL ELEVATOR - go run main.go id
	elevio.Init("localhost:15657", elevio.N_FLOORS)
	elevio.LocalId = initElevSpec[1]

	// USE FOR ELEVATOR SIMULATOR - USE go run main.go id port
	//setupSimulator.SetupOneSimulator(initElevSpec)
	//elevio.LocalId = initElevSpec[2]

	fmt.Println("Elevator id: ", elevio.LocalId)

	drvButtons := make(chan elevio.ButtonEvent)
	drvFloors := make(chan int)
	drvObstacle := make(chan bool)
	drvStop := make(chan bool)
	InitializeElevatorChannel := make(chan bool)

	peerUpdateRx := make(chan peers.PeerUpdate)
	peerUpdateTx := make(chan bool)
	go peers.Transmitter(19648, elevio.LocalId, peerUpdateTx)
	go peers.Receiver(19648, peerUpdateRx)

	backupCabTx := make(chan elevio.BackupCabMsg)
	backupCabRx := make(chan elevio.BackupCabMsg)
	go bcast.Transmitter(13572, backupCabTx)
	go bcast.Receiver(13572, backupCabRx)

	ElevatorInfoWithHallAndIDTx := make(chan costfunc.ElevatorInfoToBroadcast)
	ElevatorInfoWithHallAndIDRx := make(chan costfunc.ElevatorInfoToBroadcast)
	go bcast.Transmitter(13576, ElevatorInfoWithHallAndIDTx)
	go bcast.Receiver(13576, ElevatorInfoWithHallAndIDRx)

	go elevio.StartPollButtonsAndSensors(drvButtons, drvFloors, drvObstacle, drvStop)
	go orders.RegisterNewOrders(drvButtons)
	go finiteStateMachine.RunElevatorFSM(drvFloors, drvObstacle, drvStop, InitializeElevatorChannel, ElevatorInfoWithHallAndIDTx)
	go communication.ElevatorCommunication(ElevatorInfoWithHallAndIDRx, backupCabTx, backupCabRx, InitializeElevatorChannel, peerUpdateRx)

	for {
		select {}
	}
}
