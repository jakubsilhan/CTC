package main

import (
	"fmt"
	"goenv/Services"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"sync"
	"time"
)

// Variables

// Synchronization

var end sync.WaitGroup

// Initializations

// Config is a struct for program configuration
type Config struct {
	Cars struct {
		Count          int `yaml:"count"`
		ArrivalTimeMin int `yaml:"arrival_time_min"`
		ArrivalTimeMax int `yaml:"arrival_time_max"`
	} `yaml:"cars"`
	Stations struct {
		Gas struct {
			Count        int `yaml:"count"`
			ServeTimeMin int `yaml:"serve_time_min"`
			ServeTimeMax int `yaml:"serve_time_max"`
		} `yaml:"gas"`
		Diesel struct {
			Count        int `yaml:"count"`
			ServeTimeMin int `yaml:"serve_time_min"`
			ServeTimeMax int `yaml:"serve_time_max"`
		} `yaml:"diesel"`
		Lpg struct {
			Count        int `yaml:"count"`
			ServeTimeMin int `yaml:"serve_time_min"`
			ServeTimeMax int `yaml:"serve_time_max"`
		} `yaml:"lpg"`
		Electric struct {
			Count        int `yaml:"count"`
			ServeTimeMin int `yaml:"serve_time_min"`
			ServeTimeMax int `yaml:"serve_time_max"`
		} `yaml:"electric"`
	} `yaml:"stations"`
	Registers struct {
		Count         int `yaml:"count"`
		HandleTimeMin int `yaml:"handle_time_min"`
		HandleTimeMax int `yaml:"handle_time_max"`
	} `yaml:"registers"`
}

// loadConfigFile loads configuration from yaml into set variables
func loadConfigFile() {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading config.yaml file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		log.Fatalf("Error unmarshalling config.yaml file: %v", err)
	}

	// Loads the configuration into existing variables
	Services.StaggerMin = config.Cars.ArrivalTimeMin
	Services.StaggerMax = config.Cars.ArrivalTimeMax
	Services.CarNum = config.Cars.Count
	Services.NumGas = config.Stations.Gas.Count
	Services.GasMinT = config.Stations.Gas.ServeTimeMin
	Services.GasMaxT = config.Stations.Gas.ServeTimeMax
	Services.NumDiesel = config.Stations.Diesel.Count
	Services.DieselMinT = config.Stations.Diesel.ServeTimeMin
	Services.DieselMaxT = config.Stations.Diesel.ServeTimeMax
	Services.NumLPG = config.Stations.Lpg.Count
	Services.LpgMinT = config.Stations.Lpg.ServeTimeMin
	Services.LpgMaxT = config.Stations.Lpg.ServeTimeMax
	Services.NumElectric = config.Stations.Electric.Count
	Services.ElectricMinT = config.Stations.Electric.ServeTimeMin
	Services.ElectricMaxT = config.Stations.Electric.ServeTimeMax
	Services.NumRegisters = config.Registers.Count
	Services.MinPaymentT = config.Registers.HandleTimeMin
	Services.MaxPaymentT = config.Registers.HandleTimeMax
}

// Routines

// main controls the whole simulation
func main() {
	loadConfigFile()
	//totalStandCount := Services.NumGas + Services.NumDiesel + Services.NumLPG + Services.NumElectric
	//Services.StandCreationWaiter.Add(totalStandCount)
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
	Services.StandCreationWaiter.Add(standCount)
	for _, stand := range stands {
		go Services.StandRoutine(stand)
	}
	Services.StandCreationWaiter.Wait()
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
	Services.StandFinishWaiter.Wait()
	//Services.StandFinishWaiter.Wait()
	close(Services.BuildingQueue)

	Services.RegisterWaiter.Wait()
	close(Services.Exit)

	end.Wait()
}

// StationStats is a struct for output yaml construction
type StationStats struct {
	TotalCars    int `yaml:"total_cars"`
	TotalTime    int `yaml:"total_time"`
	AvgQueueTime int `yaml:"avg_queue_time"`
	MaxQueueTime int `yaml:"max_queue_time"`
}

// FinalStats is a struct for output yaml construction
type FinalStats struct {
	Gas       StationStats `yaml:"Gas"`
	Diesel    StationStats `yaml:"Diesel"`
	LPG       StationStats `yaml:"LPG"`
	Electric  StationStats `yaml:"Electric"`
	Registers StationStats `yaml:"Registers"`
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
			//totalGasTime += car.TotalTime
			totalGasTime += car.FuelTime
			totalGasQueue += car.StandQueueTime
			gasCount++
			if int(car.StandQueueTime) > maxGasQueue {
				maxGasQueue = int(car.StandQueueTime)
			}
		case Services.Diesel:
			//totalDieselTime += car.TotalTime
			totalDieselTime += car.FuelTime
			totalDieselQueue += car.StandQueueTime
			dieselCount++
			if int(car.StandQueueTime) > maxDieselQueue {
				maxDieselQueue = int(car.StandQueueTime)
			}
		case Services.LPG:
			//totalLPGTime += car.TotalTime
			totalLPGTime += car.FuelTime
			totalLPGQueue += car.StandQueueTime
			lpgCount++
			if int(car.StandQueueTime) > maxLPGQueue {
				maxLPGQueue = int(car.StandQueueTime)
			}
		case Services.Electric:
			//totalElectricTime += car.TotalTime
			totalElectricTime += car.FuelTime
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
	// Creating final yaml
	stats := FinalStats{
		Gas: StationStats{
			TotalCars:    gasCount,          // number of cars that went through this stand
			TotalTime:    int(totalGasTime), // the total time cars spent fueling on the station for this stand
			AvgQueueTime: averageGasQueue,   // average time spent in a queue for this stand
			MaxQueueTime: maxGasQueue,       // max time spent in a queue for this stand
		},
		Diesel: StationStats{
			TotalCars:    dieselCount,
			TotalTime:    int(totalDieselTime),
			AvgQueueTime: averageDieselQueue,
			MaxQueueTime: maxDieselQueue,
		},
		LPG: StationStats{
			TotalCars:    lpgCount,
			TotalTime:    int(totalLPGTime),
			AvgQueueTime: averageLPGQueue,
			MaxQueueTime: maxLPGQueue,
		},
		Electric: StationStats{
			TotalCars:    electricCount,
			TotalTime:    int(totalElectricTime),
			AvgQueueTime: averageElectricQueue,
			MaxQueueTime: maxElectricQueue,
		},
		Registers: StationStats{
			TotalCars:    totalCars,              // number of cars that went through payment
			TotalTime:    int(totalRegisterTime), // total time spent at the register
			AvgQueueTime: averageRegisterQueue,   // average time spent in the queues for the registers
			MaxQueueTime: maxRegisterQueue,       // max time spent in the queues for the registers
		},
	}
	yamlStats, err := yaml.Marshal(&stats)
	if err != nil {
		log.Fatalf("Error marshalling stats to YAML: %v", err)
	}

	err = os.WriteFile("final_stats.yaml", yamlStats, 0777)
	if err != nil {
		log.Fatalf("Error writing YAML to file: %v", err)
	}

	fmt.Printf("Final statistics:\n%s\n", string(yamlStats))

	end.Done()
}
