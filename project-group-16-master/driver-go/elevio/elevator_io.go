package elevio

import (
	"fmt"
	"net"
	"sync"
	"time"
)

const _pollRate = 20 * time.Millisecond

var _initialized bool = false
var _numFloors int = 4
var _numButtons int = 3
var _mtx sync.Mutex
var _conn net.Conn

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down MotorDirection = -1
	MD_Stop MotorDirection = 0
)

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

const (
	N_FLOORS  int = 4
	N_BUTTONS int = 3
)

const (
	NewOrder       int = 0
	ConfirmedOrder int = 1
	CompletedOrder int = 2
)

type ElevatorType int

const (
	NewElevator  int = 0
	LostElevator int = 1
)

type BackupCabMsg struct {
	BackupCabOrders [N_FLOORS]bool
	Id              string
}

type SyncHallOrdersMsg struct {
	SyncHallOrders [N_BUTTONS - 1][N_FLOORS]int
	Id             string
}

type PeerUpdateMsg struct {
	IdsPeerUpdate  []string
	TypeOfElevator int
}

var LocalId string

type NumberOfActiveElevators struct {
	mu     sync.Mutex
	number int
}

var numberOfActiveElevators NumberOfActiveElevators

func UpdateNumberOfActiveElevators(number int) {
	numberOfActiveElevators.mu.Lock()
	numberOfActiveElevators.number = number
	numberOfActiveElevators.mu.Unlock()
}

func GetNumberOfActiveElevators() int {
	numberOfActiveElevators.mu.Lock()
	copyNumberOfElevators := numberOfActiveElevators.number
	numberOfActiveElevators.mu.Unlock()
	return copyNumberOfElevators
}

func StartPollButtonsAndSensors(
	drvButtons chan<- ButtonEvent,
	drvFloors chan<- int,
	drvObstacle chan<- bool,
	drvStop chan<- bool,
) {
	go PollButtons(drvButtons)
	go PollFloorSensor(drvFloors)
	go PollObstructionSwitch(drvObstacle)
	go PollStopButton(drvStop)
}

func Init(addr string, numFloors int) {
	if _initialized {
		fmt.Println("Driver already initialized!")
		return
	}
	_numFloors = numFloors
	_mtx = sync.Mutex{}
	var err error
	_conn, err = net.Dial("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	_initialized = true
}

func SetMotorDirection(dir MotorDirection) {
	write([4]byte{1, byte(dir), 0, 0})
}

func SetButtonLamp(button ButtonType, floor int, value bool) {
	write([4]byte{2, byte(button), byte(floor), toByte(value)})
}

func SetFloorIndicator(floor int) {
	write([4]byte{3, byte(floor), 0, 0})
}

func SetDoorOpenLamp(value bool) {
	write([4]byte{4, toByte(value), 0, 0})
}

func SetStopLamp(value bool) {
	write([4]byte{5, toByte(value), 0, 0})
}

func PollButtons(receiver chan<- ButtonEvent) {
	prev := make([][3]bool, _numFloors)
	for {
		time.Sleep(_pollRate)
		for f := 0; f < _numFloors; f++ {
			for b := ButtonType(0); b < 3; b++ {
				v := GetButton(b, f)
				if v != prev[f][b] && v != false {
					receiver <- ButtonEvent{f, ButtonType(b)}
				}
				prev[f][b] = v
			}
		}
	}
}

func PollFloorSensor(receiver chan<- int) {
	prev := -1
	for {
		time.Sleep(_pollRate)
		v := GetFloor()
		if v != prev && v != -1 {
			receiver <- v
		}
		prev = v
	}
}

func PollStopButton(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetStop()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func PollObstructionSwitch(receiver chan<- bool) {
	prev := false
	for {
		time.Sleep(_pollRate)
		v := GetObstruction()
		if v != prev {
			receiver <- v
		}
		prev = v
	}
}

func GetButton(button ButtonType, floor int) bool {
	a := read([4]byte{6, byte(button), byte(floor), 0})
	return toBool(a[1])
}

func GetFloor() int {
	a := read([4]byte{7, 0, 0, 0})
	if a[1] != 0 {
		return int(a[2])
	} else {
		return -1
	}
}

func GetStop() bool {
	a := read([4]byte{8, 0, 0, 0})
	return toBool(a[1])
}

func GetObstruction() bool {
	a := read([4]byte{9, 0, 0, 0})
	return toBool(a[1])
}

func read(in [4]byte) [4]byte {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	var out [4]byte
	_, err = _conn.Read(out[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}

	return out
}

func write(in [4]byte) {
	_mtx.Lock()
	defer _mtx.Unlock()

	_, err := _conn.Write(in[:])
	if err != nil {
		panic("Lost connection to Elevator Server")
	}
}

func toByte(a bool) byte {
	var b byte = 0
	if a {
		b = 1
	}
	return b
}

func toBool(a byte) bool {
	var b bool = false
	if a != 0 {
		b = true
	}
	return b
}

//ekstra

func TurnOffAllLights() {
	for floor := 0; floor < _numFloors; floor++ {
		for button := ButtonType(0); button < ButtonType(_numButtons); button++ {
			SetButtonLamp(button, floor, false)
		}
	}
}
