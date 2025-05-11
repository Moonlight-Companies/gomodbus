package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Moonlight-Companies/gomodbus/client"
	"github.com/Moonlight-Companies/gomodbus/common"
	"github.com/Moonlight-Companies/gomodbus/logging"
	"github.com/Moonlight-Companies/gomodbus/transport"
)

func readLoop(name string, modbusClient *client.TCPClient) {
	readNumber := 0
	prevValues := make(map[int]interface{})

	sum := time.Duration(0)
	count := 0
	lastReport := time.Now()
	for {
		time.Sleep(time.Millisecond * 5)

		ta := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		discreteInputs, err := modbusClient.ReadDiscreteInputs(ctx, 0, 100)
		if err != nil {
			log.Printf("%s: Failed to read discrete inputs: %v", name, err)
			cancel()
			return
		}
		readNumber++

		if time.Since(ta) > 25*time.Millisecond {
			fmt.Printf("%s: %v: Discrete inputs read in %v\n", name, readNumber, time.Since(ta))
		}

		changes := 0
		for i, value := range discreteInputs {
			if prev, ok := prevValues[i]; ok {
				if prev != value {
					//fmt.Printf("Change DI %d changed from %v to %v\n", i, prev, value)
					changes++
				}
			} else {
				//fmt.Printf("Discover DI %d: %v\n", i, value)
				changes++
			}
			prevValues[i] = value
		}

		count++
		sum += time.Since(ta)

		if time.Since(lastReport) > 5*time.Second {
			avg := sum / time.Duration(count)
			fmt.Printf("%s: %v: Average time for %v reads: %v\n", name, readNumber, count, avg)

			sum = 0
			count = 0
			lastReport = time.Now()
		}

		cancel()
	}
}

func main() {
	// Create a logger
	logger := logging.NewLogger(
		logging.WithLevel(common.LevelInfo),
	)

	ip := "10.2.163.36"
	//ip := "10.2.163.32"

	//ip := "127.0.0.1"
	//port := 5022

	for {
		fmt.Println("Starting Modbus TCP client...", ip)
		// Create a new client with options
		modbusClient := client.NewTCPClient(
			ip, // Server address
			//transport.WithPort(port), // Server port (default: 502)
			transport.WithTimeoutOption(5*time.Second), // Timeout (default: 30s)
			transport.WithTransportLogger(logger),
		).WithOptions(
			client.WithTCPLogger(logger),
		)

		// Connect to the server
		ctx := context.Background()
		err := modbusClient.Connect(ctx)
		if err != nil {
			fmt.Println("Failed to connect to Modbus server:", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Set the unit ID (slave ID)
		modbusClient.WithUnitID(common.UnitID(1))

		// Block on the read loop
		wg := sync.WaitGroup{}
		threads := 1
		wg.Add(threads)
		for i := range threads {
			time.Sleep(time.Millisecond * 17)
			go func() {
				defer wg.Done()
				name := fmt.Sprintf("loop%d", i)
				readLoop(name, modbusClient)
				fmt.Println("Read loop", name, "exited")
			}()
		}
		wg.Wait()
		fmt.Println("Read loop exited, disconnecting...")

		modbusClient.Disconnect(ctx)

		fmt.Println("Disconnected from Modbus server, retrying...")
	}
}
