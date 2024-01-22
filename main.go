package main

import (
	"flag"
	"fmt"
	"github.com/uoosef/wireguard-go/device"
	"github.com/uoosef/wireguard-go/warp"
	"github.com/uoosef/wireguard-go/wiresocks"
	"log"
)

func usage() {
	fmt.Println("Usage: wiresocks [-v] [-b addr:port] [-l license] <config file path>")
	flag.PrintDefaults()
}

func main() {
	var (
		verbose     = flag.Bool("v", false, "verbose")
		bindAddress = flag.String("b", "127.0.0.1:8086", "socks bind address")
		configFile  = flag.String("c", "./wgcf-profile.ini", "ini config file path")
		endpoint    = flag.String("e", "162.159.195.1:8854", "warp clean ip")
		license     = flag.String("k", "", "license key")
	)

	flag.Usage = usage
	flag.Parse()

	// check if wgcf-profile.conf exists
	if !warp.CheckProfileExists(*license) {
		warp.LoadOrCreateIdentity(*license, *endpoint)
	}

	conf, err := wiresocks.ParseConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}

	logLevel := device.LogLevelVerbose
	if !*verbose {
		logLevel = device.LogLevelSilent
	}

	tnet, err := wiresocks.StartWireguard(conf.Device, logLevel)
	if err != nil {
		log.Fatal(err)
	}

	go wiresocks.StartProxy(tnet, *bindAddress)

	fmt.Println("Wiresocks started successfully")

	select {}
}
