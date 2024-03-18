package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

type FuelType string

const (
	Gas      FuelType = "gas"
	Diesel   FuelType = "diesel"
	Electric FuelType = "electric"
	LPG      FuelType = "lpg"
)

type Car struct {
	ID   int
	Fuel FuelType
}

type Stats struct {
	totalCars    int
	totalTime    int
	avgQueueTime int
	maxQueueTime int
}

type carStats struct {
	fuel      FuelType
	queueTime int
	fuelTime  int
	serveTime int
}

// initial variables
var wg sync.WaitGroup
var wgArrival sync.WaitGroup

// Station queue
var staggerMin = 2
var staggerMax = 6
var carNum = 100

// stands
var numGas = 2
var gasStand = make(chan int, numGas)
var gasMinT = 1
var gasMaxT = 2
var numDiesel = 2
var dieselStand = make(chan int, numDiesel)
var dieselMinT = 1
var dieselMaxT = 2
var numLPG = 1
var LPGStand = make(chan int, numLPG)
var lpgMinT = 1
var lpgMaxT = 2
var numElectric = 1
var electricStand = make(chan int, numElectric)
var electricMinT = 1
var electricMaxT = 2

// registers
var numRegisters = 3
var availableCashRegister = make(chan int, numRegisters)
var minPayment = 1
var maxPayment = 7

// aggregation
var processedCars = make(chan carStats, carNum)

func loadEnv() {
	err := godotenv.Load("setup.env")

	if err != nil {
		log.Fatalf("Error loading setup.env file")
	}
	// car creation
	staggerMin = loadIntEnvVariable("ARRIVAL_MIN")
	staggerMax = loadIntEnvVariable("ARRIVAL_MAX")
	carNum = loadIntEnvVariable("CAR_COUNT")
	// gas
	numGas = loadIntEnvVariable("GAS_COUNT")
	gasMinT = loadIntEnvVariable("GAS_SERVE_TIME_MIN")
	gasMaxT = loadIntEnvVariable("GAS_SERVE_TIME_MAX")
	// diesel
	numDiesel = loadIntEnvVariable("DIESEL_COUNT")
	dieselMinT = loadIntEnvVariable("DIESEL_SERVE_TIME_MIN")
	dieselMaxT = loadIntEnvVariable("DIESEL_SERVE_TIME_MAX")
	// lpg
	numLPG = loadIntEnvVariable("LPG_COUNT")
	lpgMinT = loadIntEnvVariable("LPG_SERVE_TIME_MIN")
	lpgMaxT = loadIntEnvVariable("LPG_SERVE_TIME_MAX")
	// electric
	numElectric = loadIntEnvVariable("ELECTRIC_COUNT")
	electricMinT = loadIntEnvVariable("ELECTRIC_SERVE_TIME_MIN")
	electricMaxT = loadIntEnvVariable("ELECTRIC_SERVE_TIME_MAX")
	// registers
	numRegisters = loadIntEnvVariable("REGISTER_COUNT")
	minPayment = loadIntEnvVariable("REGISTER_HANDLE_TIME_MIN")
	maxPayment = loadIntEnvVariable("REGISTER_HANDLE_TIME_MAX")
	// Recreating channels with updated capacities
	gasStand = make(chan int, numGas)
	dieselStand = make(chan int, numDiesel)
	LPGStand = make(chan int, numLPG)
	electricStand = make(chan int, numElectric)
	availableCashRegister = make(chan int, numRegisters)
}

func loadIntEnvVariable(key string) int {
	number, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Fatalf("Error loading environment variable: " + key)
	}
	return number
}

func main() {
	loadEnv()
	// Initialize stands
	for i := 0; i < numGas; i++ {
		gasStand <- 1
	}
	for i := 0; i < numDiesel; i++ {
		dieselStand <- 1
	}
	for i := 0; i < numLPG; i++ {
		LPGStand <- 1
	}
	for i := 0; i < numElectric; i++ {
		electricStand <- 1
	}
	// Initialize cash registers
	for i := 0; i < numRegisters; i++ {
		availableCashRegister <- 1
	}
	go stationQueue()
	// Sleep to allow for all cars to arrive
	time.Sleep(1 * time.Second)
	wgArrival.Wait()
	wg.Wait()
}

func stationQueue() {
	wgArrival.Add(1)
	defer wgArrival.Done()

	// Car arrivals
	for i := 0; i < carNum; i++ {
		go carRoutine(Car{ID: i, Fuel: genFuelType()})
		stagger := time.Duration(rand.Intn(staggerMax-staggerMin) + staggerMin)
		time.Sleep(stagger * time.Second)
	}
}

func carRoutine(car Car) {
	wg.Add(1)
	defer wg.Done()
	// Routine timer starts
	startQueueT := time.Now()
	var currentStand chan int
	var refuelTime time.Duration = 0
	// Starts occupying fuel stand
	switch car.Fuel {
	case Gas:
		currentStand = gasStand
		refuelTime = time.Duration(rand.Intn(gasMaxT-gasMinT) + gasMinT)
	case Diesel:
		currentStand = dieselStand
		refuelTime = time.Duration(rand.Intn(dieselMaxT-dieselMinT) + dieselMinT)
	case LPG:
		currentStand = LPGStand
		refuelTime = time.Duration(rand.Intn(lpgMaxT-lpgMinT) + lpgMinT)
	case Electric:
		currentStand = electricStand
		refuelTime = time.Duration(rand.Intn(electricMaxT-electricMinT) + electricMinT)
	default:
		currentStand = gasStand
		refuelTime = time.Duration(rand.Intn(gasMaxT-gasMinT) + gasMinT)
	}
	<-currentStand
	queueT := int(time.Since(startQueueT).Seconds())
	// Refueling
	time.Sleep(refuelTime * time.Second)
	fuelTime := int(refuelTime.Seconds())
	// Attends cash register
	<-availableCashRegister
	payTime := payForFuel()
	// Leaves station
	availableCashRegister <- 1
	currentStand <- 1
	// Prints car info
	totalTime := int(time.Since(startQueueT).Seconds())
	fmt.Printf("A %s Car number %d waited %d refueled in %d seconds and paid in %d seconds for a total of % d seconds\n", car.Fuel, car.ID, queueT, refuelTime, payTime, totalTime)
	processedCars <- carStats{fuel: car.Fuel, queueTime: queueT, fuelTime: fuelTime, serveTime: payTime}
}

// genFuelType returns a random fuel type.
func genFuelType() FuelType {
	fuelTypes := []FuelType{Gas, Diesel, Electric, LPG}
	randomIndex := rand.Intn(len(fuelTypes))
	return fuelTypes[randomIndex]
}

func payForFuel() int {
	randomNumber := rand.Intn(maxPayment-minPayment) + minPayment
	randTime := time.Duration(randomNumber) * time.Second
	time.Sleep(randTime)
	return randomNumber
}
