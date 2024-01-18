package finiteStateMachine

import (
	"fmt"
	"time"

	"driver-go/costfunc"
	"driver-go/elevio"
	"driver-go/orders"
)

type ElevatorState int

const (
	STATE_IDLE      ElevatorState = 0
	STATE_MOVING    ElevatorState = 1
	STATE_DOOR_OPEN ElevatorState = 2
)

const killTime = 20
const immobileTime = 12

func InitFiniteStateMachine(InitializeElevatorChannel chan<- bool) {
	elevio.SetStopLamp(false)
	elevio.SetDoorOpenLamp(false)
	elevio.TurnOffAllLights()

	if elevio.GetFloor() == -1 {
		elevio.SetMotorDirection(elevio.MD_Down)
	}
	for elevio.GetFloor() == -1 {

	}
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevio.SetFloorIndicator(elevio.GetFloor())

	InitializeElevatorChannel <- true
}

func calculateDirection(
	prevFloor int,
	prevMotorDirection elevio.MotorDirection,
) elevio.MotorDirection {
	switch prevMotorDirection {
	case elevio.MD_Up:
		if orders.CheckForAnyLocalOrderAbovePrevFloor(prevFloor) {
			return elevio.MD_Up
		} else if orders.CheckForAnyLocalOrderBelowPrevFloor(prevFloor) {
			return elevio.MD_Down
		}
	case elevio.MD_Down:
		if orders.CheckForAnyLocalOrderBelowPrevFloor(prevFloor) {
			return elevio.MD_Down
		} else if orders.CheckForAnyLocalOrderAbovePrevFloor(prevFloor) {
			return elevio.MD_Up
		}
	}
	return elevio.MD_Stop
}

func handleStopButton(
	stop bool,
) {
	fmt.Printf("Stop: %+v\n", stop)
	elevio.SetMotorDirection(elevio.MD_Stop)
}

func stopAndClearOrder(button elevio.ButtonType, currentFloor int) {
	elevio.SetMotorDirection(elevio.MD_Stop)
	elevio.SetDoorOpenLamp(true)

	orders.SetLocalOrder(currentFloor, button, false)
	elevio.SetButtonLamp(button, currentFloor, false)

	if button != elevio.BT_Cab {
		orders.SetGlobalHallOrders(currentFloor, button, false)
		orders.SetHallStatusForSync(currentFloor, button, orders.NoOrder)
	}
}

func shouldElevatorStop(currentFloor int, prevMotorDirection elevio.MotorDirection) bool {
	localOrders := orders.GetLocalOrders()

	switch prevMotorDirection {
	case elevio.MD_Down:
		if localOrders[elevio.BT_HallDown][currentFloor] == true || localOrders[elevio.BT_Cab][currentFloor] == true {
			if localOrders[elevio.BT_HallDown][currentFloor] == true && localOrders[elevio.BT_Cab][currentFloor] == true {
				stopAndClearOrder(elevio.BT_HallDown, currentFloor)
				stopAndClearOrder(elevio.BT_Cab, currentFloor)
			} else if localOrders[elevio.BT_HallDown][currentFloor] == true {
				stopAndClearOrder(elevio.BT_HallDown, currentFloor)
			} else if localOrders[elevio.BT_Cab][currentFloor] == true {
				stopAndClearOrder(elevio.BT_Cab, currentFloor)
				if localOrders[elevio.BT_HallUp][currentFloor] == true {
					stopAndClearOrder(elevio.BT_HallUp, currentFloor)
				}
			}
			return true
		} else if orders.CheckForAnyLocalOrderBelowPrevFloor(currentFloor) == false {
			stopAndClearOrder(elevio.BT_HallUp, currentFloor)
			return true
		}

	case elevio.MD_Up:
		if localOrders[elevio.BT_HallUp][currentFloor] == true || localOrders[elevio.BT_Cab][currentFloor] == true {
			if localOrders[elevio.BT_HallUp][currentFloor] == true && localOrders[elevio.BT_Cab][currentFloor] == true {
				stopAndClearOrder(elevio.BT_HallUp, currentFloor)
				stopAndClearOrder(elevio.BT_Cab, currentFloor)
			} else if localOrders[elevio.BT_HallUp][currentFloor] == true {
				stopAndClearOrder(elevio.BT_HallUp, currentFloor)
			} else if localOrders[elevio.BT_Cab][currentFloor] == true {
				stopAndClearOrder(elevio.BT_Cab, currentFloor)
				if localOrders[elevio.BT_HallDown][currentFloor] == true {
					stopAndClearOrder(elevio.BT_HallDown, currentFloor)
				}
			}
			return true
		} else if orders.CheckForAnyLocalOrderAbovePrevFloor(currentFloor) == false {
			stopAndClearOrder(elevio.BT_HallDown, currentFloor)
			return true
		}
	}
	return false
}

func orderAtThisFloor(currentFloor int) bool {
	for button := 0; button < elevio.N_BUTTONS; button++ {
		if orders.GetLocalOrders()[button][currentFloor] == true {
			return true
		}
	}
	return false
}

func RunElevatorFSM(
	drvFloors <-chan int,
	drvObstacle <-chan bool,
	drvStop <-chan bool,
	InitializeElevatorChannel chan<- bool,
	InfoStructWithHallAndIDTx chan<- costfunc.ElevatorInfoToBroadcast,
) {
	InitFiniteStateMachine(InitializeElevatorChannel)

	var prevFloor int = <-drvFloors
	fmt.Print("Init at floor: ", prevFloor, "\n")
	var obstacle bool = false
	var prevMotorDir elevio.MotorDirection = elevio.MD_Down
	var currentState = STATE_IDLE

	doorTimer := time.NewTimer(3 * time.Second)
	doorTimer.Stop()
	orderTimer := time.NewTimer(3 * time.Second)
	orderTimer.Stop()
	motorProblemBetweenFloorstimer := time.NewTimer(immobileTime * time.Second)
	motorProblemBetweenFloorstimer.Stop()

	var localStateStruct costfunc.HRAElevState
	var localElevatorStruct costfunc.ElevatorInfoToBroadcast

	localStateStruct.Behavior = "idle"
	localStateStruct.Floor = prevFloor
	localStateStruct.Direction = "stop"

	localStateStruct.CabRequests = orders.GetLocalCabOrders()
	localElevatorStruct.ElevState = localStateStruct
	localElevatorStruct.Id = elevio.LocalId
	localElevatorStruct.SyncHall = orders.GetHallStatusForSync()

	for {
		select {
		case obstacle = <-drvObstacle:
			fmt.Println(obstacle)

		case stop := <-drvStop:
			handleStopButton(stop)

		case prevFloor = <-drvFloors:
			fmt.Println("Have arrived at floor ", prevFloor)
			elevio.SetFloorIndicator(prevFloor)
			motorProblemBetweenFloorstimer = time.NewTimer(immobileTime * time.Second)

			localStateStruct.Floor = prevFloor

			if shouldElevatorStop(prevFloor, prevMotorDir) == true {
				localStateStruct.Direction = "stop"

				println("We stopped here, prevmotordircetion was", prevMotorDir)

				currentState = STATE_DOOR_OPEN
				localStateStruct.Behavior = "doorOpen"
				orderTimer = time.NewTimer(killTime * time.Second)
				motorProblemBetweenFloorstimer.Stop()

				doorTimer = time.NewTimer(2 * time.Second)
				localStateStruct.Direction = "stop"
			}

		case <-doorTimer.C:
			if currentState == STATE_DOOR_OPEN {
				if obstacle == true {
					doorTimer = time.NewTimer(2 * time.Second)
				} else {
					elevio.SetDoorOpenLamp(false)
					currentState = STATE_IDLE
					orderTimer = time.NewTimer(killTime * time.Second)
					localStateStruct.Behavior = "moving"
				}
			}

		case <-orderTimer.C:
			fmt.Println("Can't handle order")
			panic(1)
			//Assumption: manuel restart allowed

		case <-motorProblemBetweenFloorstimer.C:
			fmt.Println("Motorproblems!")
			fmt.Println("Restart should be considered")

		default:
			if orders.CheckForAnyLocalOrder() == true && currentState != STATE_DOOR_OPEN {
				if currentState == STATE_IDLE {
					orderTimer = time.NewTimer(killTime * time.Second)
					motorProblemBetweenFloorstimer = time.NewTimer(immobileTime * time.Second)
				}
				if orderAtThisFloor(prevFloor) == true && currentState == STATE_IDLE {
					if shouldElevatorStop(prevFloor, prevMotorDir) == false {
						if prevMotorDir == elevio.MD_Down {
							elevio.SetMotorDirection(elevio.MD_Down)
							currentState = STATE_MOVING
						} else if prevMotorDir == elevio.MD_Up {
							elevio.SetMotorDirection(elevio.MD_Up)
							currentState = STATE_MOVING
						}
					} else {
						println("Order at same floor handled, currently at: ", prevFloor)
						currentState = STATE_DOOR_OPEN
						orderTimer = time.NewTimer(killTime * time.Second)
						localStateStruct.Behavior = "doorOpen"
						motorProblemBetweenFloorstimer.Stop()
						doorTimer = time.NewTimer(2 * time.Second)
						localStateStruct.Direction = "stop"
					}
				} else {
					nextMotorDir := calculateDirection(prevFloor, prevMotorDir)
					if nextMotorDir != elevio.MD_Stop {
						elevio.SetMotorDirection(nextMotorDir)
						prevMotorDir = nextMotorDir
						if nextMotorDir == elevio.MD_Down {
							localStateStruct.Direction = "down"
						} else if nextMotorDir == elevio.MD_Up {
							localStateStruct.Direction = "up"
						}
						currentState = STATE_MOVING
						localStateStruct.Behavior = "moving"
					}
				}
			} else if currentState != STATE_DOOR_OPEN {
				currentState = STATE_IDLE
				localStateStruct.Behavior = "idle"
				motorProblemBetweenFloorstimer.Stop()
				orderTimer.Stop()
			}
			localStateStruct.CabRequests = orders.GetLocalCabOrders()
			localElevatorStruct.ElevState = localStateStruct
			localElevatorStruct.SyncHall = orders.GetHallStatusForSync()
			InfoStructWithHallAndIDTx <- localElevatorStruct
		}
		time.Sleep(50 * time.Millisecond)
	}
}
