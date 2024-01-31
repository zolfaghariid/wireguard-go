package main

import (
	"flag"
	"fmt"
	"github.com/uoosef/wireguard-go/psiphon"
	"github.com/uoosef/wireguard-go/warp"
	"github.com/uoosef/wireguard-go/wiresocks"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func usage() {
	log.Println("Usage: wiresocks [-v] [-b addr:port] [-l license] <config file path>")
	flag.PrintDefaults()
}

func main() {
	var (
		verbose        = flag.Bool("v", false, "verbose")
		bindAddress    = flag.String("b", "127.0.0.1:8086", "socks bind address")
		endpoint       = flag.String("e", "notset", "warp clean ip")
		license        = flag.String("k", "notset", "license key")
		country        = flag.String("country", "", "psiphon country code in ISO 3166-1 alpha-2 format")
		psiphonEnabled = flag.Bool("cfon", false, "enable psiphonEnabled over warp")
		gool           = flag.Bool("gool", false, "enable warp gooling")
	)

	flag.Usage = usage
	flag.Parse()

	// check if user input is not correct
	if (*psiphonEnabled && *gool) || (!*psiphonEnabled && *country != "") {
		log.Println("Wrong command!")
		flag.Usage()
		return
	}

	//create necessary file structures
	makeDirs()

	//create identities
	createPrimaryAndSecondaryIdentities(*license)

	//Decide Working Scenario

	if !*psiphonEnabled && !*gool {
		// just run primary warp on bindAddress
		runWarp(*bindAddress, *endpoint, "./primary/wgcf-profile.ini", *verbose, true, true)
	} else if *psiphonEnabled && !*gool {
		// run primary warp on a random tcp port and run psiphon on bind address
		runWarpWithPsiphon(*bindAddress, *endpoint, *country, *verbose)
	} else if !*psiphonEnabled && *gool {
		// run warp in warp
		runWarpInWarp(*bindAddress, *endpoint, *verbose)
	}

	//End Decide Working Scenario

	// back where you where
	if err := os.Chdir(".."); err != nil {
		log.Fatal("Error changing to 'main' directory:", err)
	}
}

func runWarp(bindAddress, endpoint, confPath string, verbose, wait bool, startProxy bool) (*wiresocks.VirtualTun, int) {
	// Setup channel to listen for interrupt signal (Ctrl+C)
	var sigChan chan os.Signal
	if wait {
		sigChan = make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	}

	conf, err := wiresocks.ParseConfig(confPath, endpoint)
	if err != nil {
		log.Fatal(err)
	}

	tnet, err := wiresocks.StartWireguard(conf.Device, verbose)
	if err != nil {
		log.Fatal(err)
	}

	if startProxy {
		go tnet.StartProxy(bindAddress)
	}

	// Wait for interrupt signal
	if wait {
		<-sigChan
	}

	return tnet, conf.Device.MTU
}

func runWarpWithPsiphon(bindAddress, endpoint, country string, verbose bool) {
	// make a random bind address for warp
	warpBindAddress, err := findFreePort("tcp")
	if err != nil {
		log.Fatal("There are no free tcp ports on Device!")
	}

	runWarp(warpBindAddress, endpoint, "./primary/wgcf-profile.ini", verbose, false, true)

	// Setup channel to listen for interrupt signal (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// run psiphon
	psiphonCtx := psiphon.RunPsiphon(warpBindAddress, bindAddress, country)

	// Wait for interrupt signal
	<-sigChan

	psiphonCtx.Done()
}

func runWarpInWarp(bindAddress, endpoint string, verbose bool) {
	// run secondary warp
	vTUN, mtu := runWarp("", endpoint, "./secondary/wgcf-profile.ini", verbose, false, false)

	// run virtual endpoint
	virtualEndpointBindAddress, err := findFreePort("udp")
	if err != nil {
		log.Fatal("There are no free udp ports on Device!")
	}

	err = wiresocks.NewVtunUDPForwarder(virtualEndpointBindAddress, "162.159.195.1:2408", vTUN, mtu+100)
	if err != nil {
		log.Fatal(err)
	}

	// run primary warp
	runWarp(bindAddress, virtualEndpointBindAddress, "./primary/wgcf-profile.ini", verbose, true, true)
}

func findFreePort(network string) (string, error) {
	if network == "udp" {
		addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return "", err
		}
		defer conn.Close()

		return conn.LocalAddr().(*net.UDPAddr).String(), nil
	}
	// Listen on TCP port 0, which tells the OS to pick a free port.
	listener, err := net.Listen(network, "127.0.0.1:0")
	if err != nil {
		return "", err // Return error if unable to listen on a port
	}
	defer listener.Close() // Ensure the listener is closed when the function returns

	// Get the port from the listener's address
	addr := listener.Addr().String()

	return addr, nil
}

func createPrimaryAndSecondaryIdentities(license string) {
	// make primary identity
	warp.UpdatePath("./primary")
	if !warp.CheckProfileExists(license) {
		err := warp.LoadOrCreateIdentity(license)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}
	// make secondary
	warp.UpdatePath("./secondary")
	if !warp.CheckProfileExists(license) {
		err := warp.LoadOrCreateIdentity(license)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}

func makeDirs() {
	stuffDir := "stuff"
	primaryDir := "primary"
	secondaryDir := "secondary"

	// Check if 'stuff' directory exists, if not create it
	if _, err := os.Stat(stuffDir); os.IsNotExist(err) {
		fmt.Println("'stuff' directory does not exist, creating it...")
		if err := os.Mkdir(stuffDir, 0755); err != nil {
			log.Fatal("Error creating 'stuff' directory:", err)
		}
	}

	// Create 'primary' and 'secondary' directories if they don't exist
	for _, dir := range []string{primaryDir, secondaryDir} {
		if _, err := os.Stat(filepath.Join(stuffDir, dir)); os.IsNotExist(err) {
			log.Printf("Creating '%s' directory...\n", dir)
			if err := os.Mkdir(filepath.Join(stuffDir, dir), 0755); err != nil {
				log.Fatalf("Error creating '%s' directory: %v\n", dir, err)
			}
		}
	}
	log.Println("'primary' and 'secondary' directories are ready")

	// Change the current working directory to 'stuff'
	if err := os.Chdir(stuffDir); err != nil {
		log.Fatal("Error changing to 'stuff' directory:", err)
	}
	log.Println("Changed working directory to 'stuff'")
}
func isPortOpen(address string, timeout time.Duration) bool {
	// Try to establish a connection
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()

	return true
}

func waitForPortToGetsOpenOrTimeout(addressToCheck string) {
	timeout := 5 * time.Second
	checkInterval := 500 * time.Millisecond

	// Set a deadline for when to stop checking
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			log.Fatalf("Timeout reached, port %s is not open", addressToCheck)
		}

		if isPortOpen(addressToCheck, checkInterval) {
			log.Printf("Port %s is now open", addressToCheck)
			break
		}

		time.Sleep(checkInterval)
	}
}
