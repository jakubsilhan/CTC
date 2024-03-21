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

// Constants for fuel types
const (
	Gas      = "gas"
	Diesel   = "diesel"
	LPG      = "LPG"
	Electric = "electric"
)

// Car represents a car arriving at the gas station
type Car struct {
	ID         int
	Fuel       FuelType
	QueueEnter time.Time
	QueueTime  time.Duration
	FuelTime   time.Duration
	PayTime    time.Duration
	carSync    *sync.WaitGroup
}

// FuelStand describes a specific stand at the station
type FuelStand struct {
	Id    int
	Type  FuelType
	Queue chan *Car
}

// CashRegister represents a cash register for payment
type CashRegister struct {
	Id    int
	Queue chan *sync.WaitGroup
}

var Exit = make(chan *Car)

func prepare() {
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
	minPaymentT = loadIntEnvVariable("REGISTER_HANDLE_TIME_MIN")
	maxPaymentT = loadIntEnvVariable("REGISTER_HANDLE_TIME_MAX")
}

func loadIntEnvVariable(key string) int {
	number, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Fatalf("Error loading environment variable: " + key)
	}
	return number
}

// stand setups
var standWaiter sync.WaitGroup
var standBuffer = 2
var registerWaiter sync.WaitGroup
var registerBuffer = 3

// stand numbers
var numGas = 2
var numDiesel = 2
var numLPG = 1
var numElectric = 1

// register numbers
var numRegisters = 2

func main() {
	prepare()
	// Creating fuel stands
	var stands []*FuelStand
	standCount := 0
	// adding gas stands
	for i := 0; i < numGas; i++ {
		stands = append(stands, newFuelStand(standCount, Gas, standBuffer))
		standCount++
	}
	// adding diesel stands
	for i := 0; i < numDiesel; i++ {
		stands = append(stands, newFuelStand(standCount, Diesel, standBuffer))
		standCount++
	}
	// adding lpg stands
	for i := 0; i < numLPG; i++ {
		stands = append(stands, newFuelStand(standCount, LPG, standBuffer))
		standCount++
	}
	//standWaiter.Add(len(stands))
	// adding electric stands
	for i := 0; i < numElectric; i++ {
		stands = append(stands, newFuelStand(standCount, Electric, standBuffer))
		standCount++
	}
	fmt.Printf("Stands: %d \n", standCount)
	// Creating registers
	//numRegisters = 4
	var registers []*CashRegister
	for i := 0; i < numRegisters; i++ {
		registers = append(registers, newCashRegister(i, registerBuffer))
	}
	fmt.Printf("Registers %d \n", numRegisters)
	end.Add(1)
	// Car creation routine
	go createCarsRoutine()
	// Stand routines
	for _, stand := range stands {
		go standRoutine(stand)
	}
	// CashRegister routines
	for _, register := range registers {
		go registerRoutine(register)
	}
	// Car shuffling routine
	go findStandRoutine(stands)
	// Register shuffling routine
	go findRegister(registers)
	// Aggregation routine
	go aggregationRoutine()

	//waitArrival.Wait()

	standWaiter.Wait()
	close(buildingQueue)

	registerWaiter.Wait()
	close(Exit)

	end.Wait()
}

// Car section
var arrivals = make(chan *Car, 20)

var staggerMax = 2
var staggerMin = 1
var carNum = 100

//var waitArrival sync.WaitGroup

// createCarsRoutine creates cars that arrive at the station
func createCarsRoutine() {
	//waitArrival.Add(1)
	for i := 0; i < carNum; i++ {
		arrivals <- &Car{ID: i, Fuel: genFuelType(), carSync: &sync.WaitGroup{}, QueueEnter: time.Now()}
		stagger := time.Duration(rand.Intn(staggerMax-staggerMin) + staggerMin)
		time.Sleep(stagger * time.Millisecond)
	}
	close(arrivals)
}

// findStand finds the best stand according to fuel type
func findStandRoutine(stands []*FuelStand) {
	for car := range arrivals {

		var bestStand *FuelStand
		bestQueueLength := -1

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
	for _, stand := range stands {
		close(stand.Queue)
	}
}

var buildingQueue = make(chan *sync.WaitGroup, 10)

// findRegister finds the best cash register for a customer
func findRegister(registers []*CashRegister) {

	for car := range buildingQueue {
		var bestRegister *CashRegister
		bestQueueLength := -1
		for _, register := range registers {
			queueLength := len(register.Queue)
			if bestQueueLength == -1 || queueLength < bestQueueLength {
				bestRegister = register
				bestQueueLength = queueLength
			}
		}
		bestRegister.Queue <- car
	}
	for _, register := range registers {
		close(register.Queue)
	}
}

// Fueling section

// newFuelStand creates a stand for specific fuel type
func newFuelStand(id int, fuel FuelType, bufferSize int) *FuelStand {
	return &FuelStand{
		Id:    id,
		Type:  fuel,
		Queue: make(chan *Car, bufferSize),
	}
}

// standRoutine runs a routine for serving cars at a stand
func standRoutine(fs *FuelStand) {
	defer standWaiter.Done()
	standWaiter.Add(1)
	fmt.Printf("Fuel stand %d is open\n", fs.Id)
	for car := range fs.Queue {
		car.QueueTime = time.Duration(time.Since(car.QueueEnter).Milliseconds())
		doFueling(car)
		car.carSync.Add(1)
		payStart := time.Now()
		buildingQueue <- car.carSync
		car.carSync.Wait()
		car.PayTime = time.Duration(time.Since(payStart).Milliseconds())
		Exit <- car
	}
	fmt.Printf("Fuel stand %d is closed\n", fs.Id)
}

// fueling times
var gasMinT = 1
var gasMaxT = 4
var dieselMinT = 2
var dieselMaxT = 5
var lpgMinT = 5
var lpgMaxT = 12
var electricMinT = 10
var electricMaxT = 21

// doFueling does fueling
func doFueling(car *Car) {
	switch car.Fuel {
	case Gas:
		car.FuelTime = randomTime(gasMinT, gasMaxT)
	case Diesel:
		car.FuelTime = randomTime(dieselMinT, dieselMaxT)
	case LPG:
		car.FuelTime = randomTime(lpgMinT, lpgMaxT)
	case Electric:
		car.FuelTime = randomTime(electricMinT, electricMaxT)
	}
	doSleeping(car.FuelTime)
}

// Payment section

// newCashRegister creates a new cash register
func newCashRegister(id, bufferSize int) *CashRegister {
	return &CashRegister{
		Id:    id,
		Queue: make(chan *sync.WaitGroup, bufferSize),
	}
}

// registerRoutine runs a routine for serving cars at a register
func registerRoutine(cs *CashRegister) {
	defer registerWaiter.Done()
	registerWaiter.Add(1)
	fmt.Printf("Cash register %d is open\n", cs.Id)
	for car := range cs.Queue {
		doPayment()
		car.Done()
	}
	fmt.Printf("Cash register %d is closed\n", cs.Id)
}

// payment times
var minPaymentT = 1
var maxPaymentT = 7

// doPayment does payment
func doPayment() {
	doSleeping(randomTime(minPaymentT, maxPaymentT))
}

var end sync.WaitGroup

// Statistics section
func aggregationRoutine() {
	var totalCars int
	var totalRegisterTime time.Duration
	var totalGasTime time.Duration
	maxGasQueue := 0
	gasCount := 0
	var totalDieselTime time.Duration
	maxDieselQueue := 0
	dieselCount := 0
	var totalLPGTime time.Duration
	maxLPGQueue := 0
	lpgCount := 0
	var totalElectricTime time.Duration
	maxElectricQueue := 0
	electricCount := 0
	for car := range Exit {
		totalCars++
		totalRegisterTime += car.PayTime
		switch car.Fuel {
		case Gas:
			totalGasTime += car.FuelTime
			gasCount++
			if car.QueueTime > 0 {
				maxGasQueue = int(car.QueueTime)
			}
		case Diesel:
			totalDieselTime += car.FuelTime
			dieselCount++
			if car.QueueTime > 0 {
				maxDieselQueue = int(car.QueueTime)
			}
		case LPG:
			totalLPGTime += car.FuelTime
			lpgCount++
			if car.QueueTime > 0 {
				maxLPGQueue = int(car.QueueTime)
			}
		case Electric:
			totalElectricTime += car.FuelTime
			electricCount++
			if car.QueueTime > 0 {
				maxElectricQueue = int(car.QueueTime)
			}
		}
		//fmt.Printf("Car %s: \n Queue: %d \n Fuel: %d \n Pay: %d \n", car.Fuel, car.QueueTime, car.FuelTime, car.PayTime)
	}
	averageGasTime := int(totalGasTime) / gasCount
	averageDieselTime := int(totalDieselTime) / dieselCount
	averageLPGTime := int(totalLPGTime) / lpgCount
	averageElectricTime := int(totalElectricTime) / electricCount
	fmt.Printf("gas:\n")
	fmt.Printf("  total_cars: %d\n", gasCount)
	fmt.Printf("  total_time: %dms\n", int(totalGasTime))
	fmt.Printf("  avg_queue_time: %dms\n", int(averageGasTime))
	fmt.Printf("  max_queue_time: %dms\n", maxGasQueue) // Since you're not calculating max queue time, I'm assuming it's a constant value.
	fmt.Printf("diesel:\n")
	fmt.Printf("  total_cars: %d\n", dieselCount)
	fmt.Printf("  total_time: %dms\n", int(totalDieselTime))
	fmt.Printf("  avg_queue_time: %dms\n", int(averageDieselTime))
	fmt.Printf("  max_queue_time: %dms\n", maxDieselQueue)
	fmt.Printf("lpg:\n")
	fmt.Printf("  total_cars: %d\n", lpgCount)
	fmt.Printf("  total_time: %dms\n", int(totalLPGTime))
	fmt.Printf("  avg_queue_time: %dms\n", int(averageLPGTime))
	fmt.Printf("  max_queue_time: %dms\n", maxLPGQueue)
	fmt.Printf("electric:\n")
	fmt.Printf("  total_cars: %d\n", electricCount)
	fmt.Printf("  total_time: %dms\n", int(totalElectricTime))
	fmt.Printf("  avg_queue_time: %dms\n", int(averageElectricTime))
	fmt.Printf("  max_queue_time: %dms\n", maxElectricQueue)

	end.Done()
}

// Utility functions

// genFuelType returns a random fuel type.
func genFuelType() FuelType {
	fuelTypes := []FuelType{Gas, Diesel, Electric, LPG}
	randomIndex := rand.Intn(len(fuelTypes))
	return fuelTypes[randomIndex]
}

// randomTime generates a random time between min and max
func randomTime(min, max int) time.Duration {
	generatedTime := time.Duration(rand.Intn(max-min) + min)
	return generatedTime
}

// doSleeping sleeps for delay * milliseconds
func doSleeping(delay time.Duration) {
	time.Sleep(delay * time.Millisecond)
}
