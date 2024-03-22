package Services

import (
	"math/rand"
	"sync"
	"time"
)

// Variables

// Arrivals
var arrivals = make(chan *Car, 20)
var StaggerMax = 2
var StaggerMin = 1
var CarNum = 100

// Initializations

type FuelType string

// Constants for fuel types
const (
	Gas      = "gas"
	Diesel   = "diesel"
	LPG      = "LPG"
	Electric = "electric"
)

// Car represents a car arriving at the gas station
type Car struct {
	ID                 int
	Fuel               FuelType
	StandQueueEnter    time.Time
	RegisterQueueEnter time.Time
	StandQueueTime     time.Duration
	RegisterQueueTime  time.Duration
	FuelTime           time.Duration
	PayTime            time.Duration
	TotalTime          time.Duration
	carSync            *sync.WaitGroup
}

// Routines

// CreateCarsRoutine creates cars that arrive at the station
func CreateCarsRoutine() {
	for i := 0; i < CarNum; i++ {
		// Adds a new car to station queue
		arrivals <- &Car{ID: i, Fuel: genFuelType(), carSync: &sync.WaitGroup{}, StandQueueEnter: time.Now()}
		// Staggers car creation
		stagger := time.Duration(rand.Intn(StaggerMax-StaggerMin) + StaggerMin)
		time.Sleep(stagger * time.Millisecond)
	}
	close(arrivals)
}
