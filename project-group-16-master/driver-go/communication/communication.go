package communication

import (
	"fmt"
	"time"

	"driver-go/costfunc"
	"driver-go/elevio"
	"driver-go/network/peers"
	"driver-go/orders"
)

func transposeMatrix(matrix [elevio.N_BUTTONS - 1][elevio.N_FLOORS]bool) [elevio.N_FLOORS][elevio.N_BUTTONS - 1]bool {

	var transposedMatrix [elevio.N_FLOORS][elevio.N_BUTTONS - 1]bool

	for floor := 0; floor < elevio.N_FLOORS; floor++ {
		for button := 0; button < elevio.N_BUTTONS-1; button++ {
			transposedMatrix[floor][button] = matrix[button][floor]
		}
	}
	return transposedMatrix
}

func ElevatorCommunication(
	ElevatorInfoWithHallAndIDRx <-chan costfunc.ElevatorInfoToBroadcast,
	backupCabTx chan<- elevio.BackupCabMsg,
	backupCabRx <-chan elevio.BackupCabMsg,
	InitializeElevatorChannel <-chan bool,
	peerUpdateCh <-chan peers.PeerUpdate,
) {
	<-InitializeElevatorChannel

	singleElevatorTimer := time.NewTimer(1 * time.Second) //Timer to make sure peers are updated before synchronization starts

	var costFuncInput costfunc.HRAInput
	costFuncInput.States = make(map[string]costfunc.HRAElevState)
	backupcommunication := make(map[string]costfunc.HRAElevState)

	numOfRequestStatuses := 0
	globalHallStatusMap := make(map[string][elevio.N_BUTTONS - 1][elevio.N_FLOORS]orders.OrderStatus)

	for {
		select {
		case <-singleElevatorTimer.C:
			if elevio.GetNumberOfActiveElevators() == 1 {
				fmt.Println("Elevator is single. Updated the Unknown status to NoOrder")
				for button := 0; button < elevio.N_BUTTONS-1; button++ {
					for floor := 0; floor < elevio.N_FLOORS; floor++ {
						orders.SetHallStatusForSync(floor, elevio.ButtonType(button), orders.NoOrder)
					}
				}
			} else {
				fmt.Println("Elevator is not single.")
			}

		case newElevatorInfo := <-ElevatorInfoWithHallAndIDRx:
			if elevio.GetNumberOfActiveElevators() == 1 {
				for button := 0; button < elevio.N_BUTTONS-1; button++ {
					for floor := 0; floor < elevio.N_FLOORS; floor++ {
						if orders.GetHallStatusForSync()[button][floor] == orders.Handling {
							orders.SetGlobalHallOrders(floor, elevio.ButtonType(button), true)
							elevio.SetButtonLamp(elevio.ButtonType(button), floor, true)
						}
					}
				}

			} else if newElevatorInfo.Id != elevio.LocalId {
				if orders.GetHallStatusForSync()[0][0] == orders.Unknown {
					fmt.Println("Elevator with Id: ", elevio.LocalId, " is beiing synced")
					for floor := 0; floor < elevio.N_FLOORS; floor++ {
						for button := 0; button < elevio.N_BUTTONS-1; button++ {
							orders.SetHallStatusForSync(floor, elevio.ButtonType(button), newElevatorInfo.SyncHall[button][floor])
							if orders.GetHallStatusForSync()[button][floor] == orders.Handling {
								orders.SetGlobalHallOrders(floor, elevio.ButtonType(button), true)
								elevio.SetButtonLamp(elevio.ButtonType(button), floor, true)
							}
						}
					}
				} else {
					globalHallStatusMap[newElevatorInfo.Id] = newElevatorInfo.SyncHall
					for button := 0; button < elevio.N_BUTTONS-1; button++ {
						for floor := 0; floor < elevio.N_FLOORS; floor++ {
							numOfRequestStatuses = 0
							for _, otherElevator := range globalHallStatusMap {
								if orders.GetHallStatusForSync()[button][floor] == orders.Requested {
									if otherElevator[button][floor] == orders.Requested || otherElevator[button][floor] == orders.Handling {
										numOfRequestStatuses++
										if numOfRequestStatuses == elevio.GetNumberOfActiveElevators()-1 {
											orders.SetGlobalHallOrders(floor, elevio.ButtonType(button), true)
											orders.SetHallStatusForSync(floor, elevio.ButtonType(button), orders.Handling)
											elevio.SetButtonLamp(elevio.ButtonType(button), floor, true)
										}
									} else {
										fmt.Println("The other elevators do not have a request for button:", button, " on floor: ", floor)
										break
									}
								} else if orders.GetHallStatusForSync()[button][floor] == orders.NoOrder && otherElevator[button][floor] == orders.Requested {
									orders.SetHallStatusForSync(floor, elevio.ButtonType(button), orders.Requested)
									fmt.Println("Order at Button: ", button, "floor: ", floor, "is being requested")
								} else if orders.GetHallStatusForSync()[button][floor] == orders.Handling && otherElevator[button][floor] == orders.NoOrder {
									fmt.Println("Order at Button: ", button, "floor: ", floor, "is completed")
									orders.SetHallStatusForSync(floor, elevio.ButtonType(button), orders.NoOrder)
									orders.SetGlobalHallOrders(floor, elevio.ButtonType(button), false)
									elevio.SetButtonLamp(elevio.ButtonType(button), floor, false)
								}
							}
						}
					}
				}
			}

			globalhallorders := orders.GetGlobalHallOrders()
			globalHallOrdersTransposed := transposeMatrix(globalhallorders)

			costFuncInput.States[newElevatorInfo.Id] = newElevatorInfo.ElevState
			costFuncInput.HallRequests = globalHallOrdersTransposed

			costfunc.RunCostFunc(costFuncInput)

		case peerUpdate := <-peerUpdateCh:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peerUpdate.Peers)
			fmt.Printf("  New:      %q\n", peerUpdate.New)
			fmt.Printf("  Lost:     %q\n", peerUpdate.Lost)
			elevio.UpdateNumberOfActiveElevators(len(peerUpdate.Peers))

			if elevio.GetNumberOfActiveElevators() <= 0 {
				elevio.UpdateNumberOfActiveElevators(1)
			}

			fmt.Print("Number of elevators: ", elevio.GetNumberOfActiveElevators(), "\n")

			if peerUpdate.New != "" {

				_, InBackup := backupcommunication[peerUpdate.New]
				if InBackup {
					fmt.Println("Fant id i backup: ", peerUpdate.New)
					fmt.Println("Data funnet: ", backupcommunication[peerUpdate.New])

					backupCabTx <- elevio.BackupCabMsg{backupcommunication[peerUpdate.New].CabRequests, peerUpdate.New}
					time.Sleep(5 * time.Millisecond)
					backupCabTx <- elevio.BackupCabMsg{backupcommunication[peerUpdate.New].CabRequests, peerUpdate.New}
					time.Sleep(5 * time.Millisecond)
					backupCabTx <- elevio.BackupCabMsg{backupcommunication[peerUpdate.New].CabRequests, peerUpdate.New}
					time.Sleep(5 * time.Millisecond)
					backupCabTx <- elevio.BackupCabMsg{backupcommunication[peerUpdate.New].CabRequests, peerUpdate.New}

					costFuncInput.States[peerUpdate.New] = backupcommunication[peerUpdate.New]
					delete(backupcommunication, peerUpdate.New)
				}
			}
			if len(peerUpdate.Lost) != 0 {

				for i := 0; i < len(peerUpdate.Lost); i++ {

					if elevio.GetNumberOfActiveElevators() == 1 {
						orders.CopyGlobalHallOrdersToLocalOrders()
					}
					backupcommunication[peerUpdate.Lost[i]] = costFuncInput.States[peerUpdate.Lost[i]]
					fmt.Println("Saving this as backup: ", backupcommunication[peerUpdate.Lost[i]])
					fmt.Println("For elevator with Id: ", peerUpdate.Lost[i])

					delete(costFuncInput.States, peerUpdate.Lost[i])
					delete(globalHallStatusMap, peerUpdate.Lost[i])
				}

			}

		case backupCabOrders := <-backupCabRx:
			fmt.Println("There is backup for elevator with Id: ", backupCabOrders.Id)

			if backupCabOrders.Id == elevio.LocalId {
				fmt.Println("Elevator: ", backupCabOrders.Id, " reviced: ")
				fmt.Println(backupCabOrders)
				for f := 0; f < elevio.N_FLOORS; f++ {
					if backupCabOrders.BackupCabOrders[f] == true {
						orders.SetLocalOrder(f, elevio.BT_Cab, true)
						elevio.SetButtonLamp(elevio.BT_Cab, f, true)
					}
				}
			}
		}
	}
}
