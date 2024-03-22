package Services

import (
	"fmt"
	"sync"
	"time"
)

// Variables

// Fueling times

var GasMinT = 1
var GasMaxT = 4
var DieselMinT = 2
var DieselMaxT = 5
var LpgMinT = 5
var LpgMaxT = 12
var ElectricMinT = 10
var ElectricMaxT = 21

// Stand setups

var StandWaiter sync.WaitGroup
var StandBuffer = 2

// Stand numbers

var NumGas = 2
var NumDiesel = 2
var NumLPG = 1
var NumElectric = 1

// Initializations

// FuelStand describes a specific stand at the station
type FuelStand struct {
	Id    int
	Type  FuelType
	Queue chan *Car
}

// NewFuelStand creates a stand for specific fuel type
func NewFuelStand(id int, fuel FuelType, bufferSize int) *FuelStand {
	return &FuelStand{
		Id:    id,
		Type:  fuel,
		Queue: make(chan *Car, bufferSize),
	}
}

// Routines

// FindStandRoutine finds the best stand according to fuel type
func FindStandRoutine(stands []*FuelStand) {
	// Station entrance queue
	for car := range arrivals {
		// Initialization
		var bestStand *FuelStand
		bestQueueLength := -1
		// Finding best stand
		for _, stand := range stands {
			if stand.Type == car.Fuel {
				queueLength := len(stand.Queue)
				if bestQueueLength == -1 || queueLength < bestQueueLength {
					bestStand = stand
					bestQueueLength = queueLength
				}
			}
		}
		bestStand.Queue <- car
	}
	// Closing all stands
	for _, stand := range stands {
		close(stand.Queue)
	}
}

// StandRoutine runs a routine for serving cars at a stand
func StandRoutine(fs *FuelStand) {
	defer StandWaiter.Done()
	StandWaiter.Add(1)
	fmt.Printf("Fuel stand %d is open\n", fs.Id)
	// Stand queue
	for car := range fs.Queue {
		car.StandQueueTime = time.Duration(time.Since(car.StandQueueEnter).Milliseconds())
		doFueling(car)
		car.carSync.Add(1)
		// Sending car to registers
		BuildingQueue <- car
		// Wait for payment to complete
		car.carSync.Wait()
	}
	fmt.Printf("Fuel stand %d is closed\n", fs.Id)
}

// Utilities

// doFueling does fueling
func doFueling(car *Car) {
	// Set fuel time according to fuel type
	switch car.Fuel {
	case Gas:
		car.FuelTime = randomTime(GasMinT, GasMaxT)
	case Diesel:
		car.FuelTime = randomTime(DieselMinT, DieselMaxT)
	case LPG:
		car.FuelTime = randomTime(LpgMinT, LpgMaxT)
	case Electric:
		car.FuelTime = randomTime(ElectricMinT, ElectricMaxT)
	}
	// Wait to finish fueling
	doSleeping(car.FuelTime)
}
