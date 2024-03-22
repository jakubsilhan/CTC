package Services

import (
	"fmt"
	"sync"
	"time"
)

// Variables

// Register setups

var RegisterWaiter sync.WaitGroup
var RegisterBuffer = 3

// Register numbers

var NumRegisters = 2

// Payment times

var MinPaymentT = 1
var MaxPaymentT = 7

// Building

var BuildingQueue = make(chan *Car, 10)

// Exit

var Exit = make(chan *Car)

// Initializations

// CashRegister represents a cash register for payment
type CashRegister struct {
	Id    int
	Queue chan *Car
}

// NewCashRegister creates a new cash register
func NewCashRegister(id, bufferSize int) *CashRegister {
	return &CashRegister{
		Id:    id,
		Queue: make(chan *Car, bufferSize),
	}
}

// Routines

// FindRegister finds the best cash register for a customer
func FindRegister(registers []*CashRegister) {
	// Station building queue
	for car := range BuildingQueue {
		var bestRegister *CashRegister
		bestQueueLength := -1
		// Finding best register
		for _, register := range registers {
			queueLength := len(register.Queue)
			if bestQueueLength == -1 || queueLength < bestQueueLength {
				bestRegister = register
				bestQueueLength = queueLength
			}
		}
		car.RegisterQueueEnter = time.Now()
		bestRegister.Queue <- car
	}
	// Closing all registers
	for _, register := range registers {
		close(register.Queue)
	}
}

// RegisterRoutine runs a routine for serving cars at a register
func RegisterRoutine(cs *CashRegister) {
	defer RegisterWaiter.Done()
	RegisterWaiter.Add(1)
	fmt.Printf("Cash register %d is open\n", cs.Id)
	// Station shop queue
	for car := range cs.Queue {
		car.RegisterQueueTime = time.Duration(time.Since(car.RegisterQueueEnter).Milliseconds())
		doPayment(car)
		// Signaling finished payment to stand
		car.carSync.Done()
		// Sending car to exit queue
		Exit <- car
	}
	fmt.Printf("Cash register %d is closed\n", cs.Id)
}

// Utilities

// doPayment does payment
func doPayment(car *Car) {
	// Generating payment time
	car.PayTime = randomTime(MinPaymentT, MaxPaymentT)
	// Waiting for payment to finish
	doSleeping(car.PayTime)
}
