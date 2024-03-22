package Services

import (
	"math/rand"
	"time"
)

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
