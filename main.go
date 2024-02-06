package main

import (
	"flag"
	"github.com/bepass-org/wireguard-go/app"
	"log"
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
		scan           = flag.Bool("scan", false, "enable warp scanner(experimental)")
	)

	flag.Usage = usage
	flag.Parse()

	app.RunWarp(*psiphonEnabled, *gool, *scan, *verbose, *country, *bindAddress, *endpoint, *license)
}
