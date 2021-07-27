package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	originalExecName = "avrdude-original"
	configFilename   = "config.json"
)

func main() {
	ownDir := getOwnDir()
	configFile := filepath.Join(ownDir, configFilename)
	originalExec := filepath.Join(ownDir, originalExecName)
	log.Println(originalExec)

	avrdudeproxy := avrdudeProxy{
		orignalExec: originalExec,
		args:        os.Args[1:],
	}

	serialPort := getPort()
	isGpioUart, err := isGpioUart(serialPort)
	if err != nil && err != errPortNotFound { // let avrdude-original handle missing port
		log.Fatal(err)
	}
	if isGpioUart {
		log.Println("GPIO UART detected. Running in GPIO reset mode")
		config, err := loadConfiguration(configFile)
		if err != nil {
			log.Fatal("Failed to load configuration file: %w", err)
		}
		avrdudeproxy.resetPin = config.DefaultResetPin
		for _, port := range config.Ports {
			if port.Name == serialPort {
				avrdudeproxy.resetPin = port.ResetPin
				break
			}
		}
		avrdudeproxy.gpioResetRun()
	} else {
		log.Println("Not GPIO UART. Running in normal mode")
		avrdudeproxy.normalRun()
	}
}

func getPort() string {
	var serailPort string
	// flag.StringVar(&serailPort, "P", "", "Destination Serial Port")
	// flag.Parse()
	if serailPort == "" {
		for _, arg := range os.Args[1:] {
			if strings.HasPrefix(arg, "-P") {
				serailPort = arg[2:]
				break
			}
		}
	}

	return serailPort
}

func getOwnDir() string {
	exec, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Dir(exec)
}
