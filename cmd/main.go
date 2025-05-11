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

	//ip := "10.2.163.36"
	//ip := "10.2.163.32"

	ip := "127.0.0.1"
	port := 5022

	for {
		fmt.Println("Starting Modbus TCP client...", ip)
		// Create a new client with options
		modbusClient := client.NewTCPClient(
			ip,                       // Server address
			transport.WithPort(port), // Server port (default: 502)
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

		// Query the server for its identity
		identity, err := modbusClient.ReadDeviceIdentification(ctx, common.ReadDeviceIDBasicStream, common.DeviceIDObjectCode(0))
		if err != nil {
			// Check if the error is due to unsupported function
			if common.IsFunctionNotSupportedError(err) {
				fmt.Println("Note: Device identification is not supported by this device")
				return
			} else {
				// It's a different kind of error
				fmt.Println("Error reading device identification:", err)
				return
			}
		} else {
			// Print basic information first
			fmt.Printf("Device Identification - %s Info:\n", common.ReadDeviceIDBasicStream)
			fmt.Println("----------------------------------")
			fmt.Printf("Vendor Name:    %s\n", identity.GetVendorName())
			fmt.Printf("Product Code:   %s\n", identity.GetProductCode())
			fmt.Printf("Revision:       %s\n", identity.GetRevision())

			// Try to get extended information if available
			extendedIdentity, err := modbusClient.ReadDeviceIdentification(ctx, common.ReadDeviceIDExtendedStream, common.DeviceIDObjectCode(0))
			if err == nil {
				fmt.Println("\nDevice Identification - Extended Info:")
				fmt.Println("-------------------------------------")

				// Display all available fields
				if vendorURL := extendedIdentity.GetVendorURL(); vendorURL != "" {
					fmt.Printf("Vendor URL:     %s\n", vendorURL)
				}
				if productName := extendedIdentity.GetProductName(); productName != "" {
					fmt.Printf("Product Name:   %s\n", productName)
				}
				if modelName := extendedIdentity.GetModelName(); modelName != "" {
					fmt.Printf("Model Name:     %s\n", modelName)
				}
				if appName := extendedIdentity.GetUserApplicationName(); appName != "" {
					fmt.Printf("App Name:       %s\n", appName)
				}

				// Display any additional objects found in extended info
				additionalObjects := false
				for _, obj := range extendedIdentity.Objects {
					// Skip standard objects
					if obj.ID <= common.DeviceIDUserAppName {
						continue
					}

					if !additionalObjects {
						fmt.Println("\nAdditional Device Objects:")
						additionalObjects = true
					}

					fmt.Printf("  Object ID 0x%02X: %s\n", byte(obj.ID), obj.Value)
				}
			} else if !common.IsFunctionNotSupportedError(err) {
				// Only print if it's not a "function not supported" error
				// as we expect some devices might only support basic identification
				fmt.Printf("\nNote: Could not retrieve extended device information: %v\n", err)
			}

			// Display raw object information for debugging
			fmt.Println("\nRaw Objects:")
			fmt.Printf("Total Objects: %d\n", identity.NumberOfObjects)
			for i, obj := range identity.Objects {
				fmt.Printf("  Object #%d: 0x%02X, Length=%d, Value='%s'\n",
					i, byte(obj.ID), obj.Length, obj.Value)
			}
		}

		// Try to get specific device identification objects
		fmt.Println("\nTrying to get specific device identification objects:")
		fmt.Println("--------------------------------------------------")

		// List of object IDs to try specifically
		specificObjects := []common.DeviceIDObjectCode{
			common.DeviceIDVendorName,         // 0x00
			common.DeviceIDProductCode,        // 0x01
			common.DeviceIDMajorMinorRevision, // 0x02
			common.DeviceIDVendorURL,          // 0x03
			common.DeviceIDProductName,        // 0x04
			common.DeviceIDModelName,          // 0x05
			common.DeviceIDUserAppName,        // 0x06
			// Try one non-standard object just to see
			common.DeviceIDObjectCode(0x80),
		}

		// Request each object individually
		for _, objectID := range specificObjects {
			specificIdentity, err := modbusClient.ReadDeviceIdentification(
				ctx,
				common.ReadDeviceIDSpecificObject,
				objectID,
			)

			if err != nil {
				if common.IsFunctionNotSupportedError(err) {
					fmt.Printf("Object 0x%02X: Not supported by device\n", byte(objectID))
				} else {
					fmt.Printf("Object 0x%02X: Error: %v\n", byte(objectID), err)
				}
			} else if specificIdentity != nil && len(specificIdentity.Objects) > 0 {
				fmt.Printf("Object 0x%02X: %s = '%s'\n",
					byte(objectID),
					objectID,
					specificIdentity.Objects[0].Value)
			} else {
				fmt.Printf("Object 0x%02X: No data returned\n", byte(objectID))
			}
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
