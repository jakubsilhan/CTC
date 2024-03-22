package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"goenv/Services"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Variables

// Synchronization

var end sync.WaitGroup

// Initializations

// loadEnvFile configures simulation according to setup.env
func loadEnvFile() {
	err := godotenv.Load("setup.env")

	if err != nil {
		log.Fatalf("Error loading setup.env file")
	}

	// car creation
	Services.StaggerMin = loadIntEnvVariable("ARRIVAL_MIN")
	Services.StaggerMax = loadIntEnvVariable("ARRIVAL_MAX")
	Services.CarNum = loadIntEnvVariable("CAR_COUNT")
	// gas
	Services.NumGas = loadIntEnvVariable("GAS_COUNT")
	Services.GasMinT = loadIntEnvVariable("GAS_SERVE_TIME_MIN")
	Services.GasMaxT = loadIntEnvVariable("GAS_SERVE_TIME_MAX")
	// diesel
	Services.NumDiesel = loadIntEnvVariable("DIESEL_COUNT")
	Services.DieselMinT = loadIntEnvVariable("DIESEL_SERVE_TIME_MIN")
	Services.DieselMaxT = loadIntEnvVariable("DIESEL_SERVE_TIME_MAX")
	// lpg
	Services.NumLPG = loadIntEnvVariable("LPG_COUNT")
	Services.LpgMinT = loadIntEnvVariable("LPG_SERVE_TIME_MIN")
	Services.LpgMaxT = loadIntEnvVariable("LPG_SERVE_TIME_MAX")
	// electric
	Services.NumElectric = loadIntEnvVariable("ELECTRIC_COUNT")
	Services.ElectricMinT = loadIntEnvVariable("ELECTRIC_SERVE_TIME_MIN")
	Services.ElectricMaxT = loadIntEnvVariable("ELECTRIC_SERVE_TIME_MAX")
	// registers
	Services.NumRegisters = loadIntEnvVariable("REGISTER_COUNT")
	Services.MinPaymentT = loadIntEnvVariable("REGISTER_HANDLE_TIME_MIN")
	Services.MaxPaymentT = loadIntEnvVariable("REGISTER_HANDLE_TIME_MAX")
	Services.StandBuffer = loadIntEnvVariable("STAND_BUFFER")
	Services.RegisterBuffer = loadIntEnvVariable("REGISTER_BUFFER")
}

// loadIntEnvVariable loads variables from .env file as an integer
func loadIntEnvVariable(key string) int {
	number, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		log.Fatalf("Error loading environment variable: " + key)
	}
	return number
}

// Routines

// main controls the whole simulation
func main() {
	loadEnvFile()
	// Creating fuel stands
	var stands []*Services.FuelStand
	standCount := 0
	// Adding gas stands
	for i := 0; i < Services.NumGas; i++ {
		stands = append(stands, Services.NewFuelStand(standCount, Services.Gas, Services.StandBuffer))
		standCount++
	}
	// Adding diesel stands
	for i := 0; i < Services.NumDiesel; i++ {
		stands = append(stands, Services.NewFuelStand(standCount, Services.Diesel, Services.StandBuffer))
		standCount++
	}
	// Adding lpg stands
	for i := 0; i < Services.NumLPG; i++ {
		stands = append(stands, Services.NewFuelStand(standCount, Services.LPG, Services.StandBuffer))
		standCount++
	}
	// Adding electric stands
	for i := 0; i < Services.NumElectric; i++ {
		stands = append(stands, Services.NewFuelStand(standCount, Services.Electric, Services.StandBuffer))
		standCount++
	}
	// Creating registers
	var registers []*Services.CashRegister
	for i := 0; i < Services.NumRegisters; i++ {
		registers = append(registers, Services.NewCashRegister(i, Services.RegisterBuffer))
	}
	end.Add(1)
	// Car creation routine
	go Services.CreateCarsRoutine()
	// Stand routines
	for _, stand := range stands {
		go Services.StandRoutine(stand)
	}
	// CashRegister routines
	for _, register := range registers {
		go Services.RegisterRoutine(register)
	}
	// Car shuffling routine
	go Services.FindStandRoutine(stands)
	// Register shuffling routine
	go Services.FindRegister(registers)
	// Aggregation routine
	go aggregationRoutine()

	// End synchronizations
	Services.StandWaiter.Wait()
	close(Services.BuildingQueue)

	Services.RegisterWaiter.Wait()
	close(Services.Exit)

	end.Wait()
}

// aggregationRoutine collects global data about the station and prints them
func aggregationRoutine() {
	var totalCars int
	var totalRegisterTime time.Duration
	var totalRegisterQueue time.Duration
	maxRegisterQueue := 0
	// Gas
	var totalGasTime time.Duration
	var totalGasQueue time.Duration
	maxGasQueue := 0
	gasCount := 0
	// Diesel
	var totalDieselTime time.Duration
	var totalDieselQueue time.Duration
	maxDieselQueue := 0
	dieselCount := 0
	// LPG
	var totalLPGTime time.Duration
	var totalLPGQueue time.Duration
	maxLPGQueue := 0
	lpgCount := 0
	// Electric
	var totalElectricTime time.Duration
	var totalElectricQueue time.Duration
	maxElectricQueue := 0
	electricCount := 0
	// Exit queue aggregates data
	for car := range Services.Exit {
		totalCars++
		totalRegisterTime += car.PayTime
		totalRegisterQueue += car.RegisterQueueTime
		car.TotalTime = time.Duration(time.Since(car.StandQueueEnter).Milliseconds())
		if int(car.RegisterQueueTime) > maxRegisterQueue {
			maxRegisterQueue = int(car.RegisterQueueTime)
		}
		switch car.Fuel {
		case Services.Gas:
			totalGasTime += car.TotalTime
			totalGasQueue += car.StandQueueTime
			gasCount++
			if int(car.StandQueueTime) > maxGasQueue {
				maxGasQueue = int(car.StandQueueTime)
			}
		case Services.Diesel:
			totalDieselTime += car.TotalTime
			totalDieselQueue += car.StandQueueTime
			dieselCount++
			if int(car.StandQueueTime) > maxDieselQueue {
				maxDieselQueue = int(car.StandQueueTime)
			}
		case Services.LPG:
			totalLPGTime += car.TotalTime
			totalLPGQueue += car.StandQueueTime
			lpgCount++
			if int(car.StandQueueTime) > maxLPGQueue {
				maxLPGQueue = int(car.StandQueueTime)
			}
		case Services.Electric:
			totalElectricTime += car.TotalTime
			totalElectricQueue += car.StandQueueTime
			electricCount++
			if int(car.StandQueueTime) > maxElectricQueue {
				maxElectricQueue = int(car.StandQueueTime)
			}
		}
		//fmt.Printf("Car %s: \n Queue: %d \n Fuel: %d \n Pay: %d \n", car.Fuel, car.StandQueueTime, car.FuelTime, car.PayTime)
	}
	// Calculating average values
	var averageGasQueue int
	if gasCount != 0 {
		averageGasQueue = int(totalGasQueue) / gasCount
	}
	var averageDieselQueue int
	if dieselCount != 0 {
		averageDieselQueue = int(totalDieselQueue) / dieselCount
	}
	var averageLPGQueue int
	if lpgCount != 0 {
		averageLPGQueue = int(totalLPGQueue) / lpgCount
	}
	var averageElectricQueue int
	if electricCount != 0 {
		averageElectricQueue = int(totalElectricQueue) / electricCount
	}
	var averageRegisterQueue int
	if totalCars != 0 {
		averageRegisterQueue = int(totalRegisterQueue) / totalCars
	}
	// Printing results
	fmt.Println("Final statistics")
	fmt.Printf("Gas:\n")
	fmt.Printf("  total_cars: %d\n", gasCount)
	fmt.Printf("  total_time: %dms\n", int(totalGasTime))
	fmt.Printf("  avg_queue_time: %dms\n", averageGasQueue)
	fmt.Printf("  max_queue_time: %dms\n", maxGasQueue)
	fmt.Printf("Diesel:\n")
	fmt.Printf("  total_cars: %d\n", dieselCount)
	fmt.Printf("  total_time: %dms\n", int(totalDieselTime))
	fmt.Printf("  avg_queue_time: %dms\n", averageDieselQueue)
	fmt.Printf("  max_queue_time: %dms\n", maxDieselQueue)
	fmt.Printf("LPG:\n")
	fmt.Printf("  total_cars: %d\n", lpgCount)
	fmt.Printf("  total_time: %dms\n", int(totalLPGTime))
	fmt.Printf("  avg_queue_time: %dms\n", averageLPGQueue)
	fmt.Printf("  max_queue_time: %dms\n", maxLPGQueue)
	fmt.Printf("Electric:\n")
	fmt.Printf("  total_cars: %d\n", electricCount)
	fmt.Printf("  total_time: %dms\n", int(totalElectricTime))
	fmt.Printf("  avg_queue_time: %dms\n", averageElectricQueue)
	fmt.Printf("  max_queue_time: %dms\n", maxElectricQueue)
	fmt.Printf("Registers:\n")
	fmt.Printf("  total_cars: %d\n", totalCars)
	fmt.Printf("  total_time: %dms\n", int(totalRegisterTime))
	fmt.Printf("  avg_queue_time: %dms\n", averageRegisterQueue)
	fmt.Printf("  max_queue_time: %dms\n", maxRegisterQueue)

	end.Done()
}
