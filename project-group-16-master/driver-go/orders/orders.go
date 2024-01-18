package orders

import (
	"sync"
	"time"

	"driver-go/elevio"
)

type OrderStatus int

const (
	Unknown   OrderStatus = 0
	NoOrder   OrderStatus = 1
	Requested OrderStatus = 2
	Handling  OrderStatus = 3
)

type HallStatusForSync struct {
	mu             sync.Mutex
	hallSyncMatrix [elevio.N_BUTTONS - 1][elevio.N_FLOORS]OrderStatus
}

type LocalOrders struct {
	mu          sync.Mutex
	localMatrix [elevio.N_BUTTONS][elevio.N_FLOORS]bool
}

type GlobalOrders struct {
	mu           sync.Mutex
	globalMatrix [elevio.N_BUTTONS - 1][elevio.N_FLOORS]bool
}

var hallStatusForSync HallStatusForSync
var globalHallOrders GlobalOrders
var localOrders LocalOrders

func GetHallStatusForSync() [elevio.N_BUTTONS - 1][elevio.N_FLOORS]OrderStatus {
	hallStatusForSync.mu.Lock()
	copyHall := hallStatusForSync.hallSyncMatrix
	hallStatusForSync.mu.Unlock()
	return copyHall
}

func SetHallStatusForSync(floor int, button elevio.ButtonType, status OrderStatus) {
	hallStatusForSync.mu.Lock()
	hallStatusForSync.hallSyncMatrix[button][floor] = status
	hallStatusForSync.mu.Unlock()
}

func GetGlobalHallOrders() [elevio.N_BUTTONS - 1][elevio.N_FLOORS]bool {
	globalHallOrders.mu.Lock()
	copyGlobal := globalHallOrders.globalMatrix
	globalHallOrders.mu.Unlock()
	return copyGlobal
}

func SetGlobalHallOrders(floor int, button elevio.ButtonType, state bool) {
	globalHallOrders.mu.Lock()
	globalHallOrders.globalMatrix[button][floor] = state
	globalHallOrders.mu.Unlock()
}

func GetLocalOrders() [elevio.N_BUTTONS][elevio.N_FLOORS]bool {
	localOrders.mu.Lock()
	copyLocal := localOrders.localMatrix
	localOrders.mu.Unlock()
	return copyLocal
}

func GetLocalCabOrders() [elevio.N_FLOORS]bool {
	localOrders.mu.Lock()
	copyLocalCab := localOrders.localMatrix[elevio.BT_Cab]
	localOrders.mu.Unlock()
	return copyLocalCab
}

func SetLocalOrder(floor int, button elevio.ButtonType, state bool) {
	localOrders.mu.Lock()
	localOrders.localMatrix[button][floor] = state
	localOrders.mu.Unlock()
}

func checkForLocalOrder(button int, floor int) bool {
	localOrders.mu.Lock()
	existOrder := localOrders.localMatrix[button][floor]
	localOrders.mu.Unlock()
	return existOrder
}

func CheckForAnyLocalOrder() bool {
	for b := 0; b < elevio.N_BUTTONS; b++ {
		for f := 0; f < elevio.N_FLOORS; f++ {
			if checkForLocalOrder(b, f) == true {
				return true
			}
		}
	}
	return false
}

func CheckForAnyLocalOrderAbovePrevFloor(prevfloor int) bool {
	for b := 0; b < elevio.N_BUTTONS; b++ {
		for f := prevfloor + 1; f < elevio.N_FLOORS; f++ {
			if GetLocalOrders()[b][f] == true {
				return true
			}
		}
	}
	return false
}

func CheckForAnyLocalOrderBelowPrevFloor(prevfloor int) bool {
	for b := 0; b < elevio.N_BUTTONS; b++ {
		for f := prevfloor - 1; f >= 0; f-- {
			if GetLocalOrders()[b][f] {
				return true
			}
		}
	}
	return false
}

func CopyGlobalHallOrdersToLocalOrders() {
	for b := 0; b < elevio.N_BUTTONS-1; b++ {
		for f := 0; f < elevio.N_FLOORS; f++ {
			if GetGlobalHallOrders()[b][f] == true {
				SetLocalOrder(f, elevio.ButtonType(b), true)
			}
		}
	}
}

func RegisterNewOrders(
	drvButtons <-chan elevio.ButtonEvent,
) {
	for {
		select {
		case buttonPressed := <-drvButtons:
			if elevio.GetNumberOfActiveElevators() == 1 {
				if buttonPressed.Button == elevio.BT_Cab {
					SetLocalOrder(buttonPressed.Floor, buttonPressed.Button, true)
					elevio.SetButtonLamp(buttonPressed.Button, buttonPressed.Floor, true)
				} else {
					println("Vi registrerer HallOrders for single elevator. Button: ", buttonPressed.Button, " Floor: ", buttonPressed.Floor)
					SetHallStatusForSync(buttonPressed.Floor, buttonPressed.Button, Handling)
				}
			} else if elevio.GetNumberOfActiveElevators() > 1 {
				if buttonPressed.Button == elevio.BT_Cab {
					SetLocalOrder(buttonPressed.Floor, buttonPressed.Button, true)
					elevio.SetButtonLamp(buttonPressed.Button, buttonPressed.Floor, true)
				} else if hallStatusForSync.hallSyncMatrix[buttonPressed.Button][buttonPressed.Floor] != Handling {
					println("Button: ", buttonPressed.Button, ", Floor: ", buttonPressed.Floor, " is pressed and request initiated")
					SetHallStatusForSync(buttonPressed.Floor, buttonPressed.Button, Requested)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}
