package wiresocks

import (
	"fmt"
	"github.com/bepass-org/proxy/pkg/mixed"
	"github.com/bepass-org/proxy/pkg/statute"
	"github.com/bepass-org/wireguard-go/tun/netstack"
	"io"
	"log"
)

// VirtualTun stores a reference to netstack network and DNS configuration
type VirtualTun struct {
	Tnet      *netstack.Net
	SystemDNS bool
	Verbose   bool
	Logger    DefaultLogger
}

type DefaultLogger struct {
	verbose bool
}

func (l DefaultLogger) Debug(v ...interface{}) {
	if l.verbose {
		log.Println(v...)
	}
}

func (l DefaultLogger) Error(v ...interface{}) {
	log.Println(v...)
}

// StartProxy spawns a socks5 server.
func (vt *VirtualTun) StartProxy(bindAddress string) {
	proxy := mixed.NewProxy(
		mixed.WithBindAddress(bindAddress),
		mixed.WithLogger(vt.Logger),
		mixed.WithUserHandler(func(request *statute.ProxyRequest) error {
			return vt.generalHandler(request)
		}),
	)
	_ = proxy.ListenAndServe()
}

func (vt *VirtualTun) generalHandler(req *statute.ProxyRequest) error {
	if vt.Verbose {
		log.Println(fmt.Sprintf("handling %s request to %s", req.Network, req.Destination))
	}
	conn, err := vt.Tnet.Dial(req.Network, req.Destination)
	if err != nil {
		return err
	}
	// Close the connections when this function exits
	defer conn.Close()
	defer req.Conn.Close()
	// Channel to notify when copy operation is done
	done := make(chan error, 1)
	// Copy data from req.Conn to conn
	go func() {
		_, err := io.Copy(conn, req.Conn)
		done <- err
	}()
	// Copy data from conn to req.Conn
	go func() {
		_, err := io.Copy(req.Conn, conn)
		done <- err
	}()
	// Wait for one of the copy operations to finish
	err = <-done
	if err != nil {
		log.Println(err)
	}
	// Close connections and wait for the other copy operation to finish
	conn.Close()
	req.Conn.Close()
	<-done

	return nil
}
