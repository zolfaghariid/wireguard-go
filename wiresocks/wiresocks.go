package wiresocks

import (
	"bytes"
	"context"
	"fmt"

	"net/netip"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/bepass-org/wireguard-go/conn"
	"github.com/bepass-org/wireguard-go/device"
	"github.com/bepass-org/wireguard-go/tun/netstack"
)

// DeviceSetting contains the parameters for setting up a tun interface
type DeviceSetting struct {
	ipcRequest string
	dns        []netip.Addr
	deviceAddr []netip.Addr
	mtu        int
}

// serialize the config into an IPC request and DeviceSetting
func createIPCRequest(conf *DeviceConfig) (*DeviceSetting, error) {
	var request bytes.Buffer

	request.WriteString(fmt.Sprintf("private_key=%s\n", conf.SecretKey))

	if conf.ListenPort != nil {
		request.WriteString(fmt.Sprintf("listen_port=%d\n", *conf.ListenPort))
	}

	for _, peer := range conf.Peers {
		request.WriteString(fmt.Sprintf(heredoc.Doc(`
				public_key=%s
				persistent_keepalive_interval=%d
				preshared_key=%s
			`),
			peer.PublicKey, peer.KeepAlive, peer.PreSharedKey,
		))
		if peer.Endpoint != nil {
			request.WriteString(fmt.Sprintf("endpoint=%s\n", *peer.Endpoint))
		}

		if len(peer.AllowedIPs) > 0 {
			for _, ip := range peer.AllowedIPs {
				request.WriteString(fmt.Sprintf("allowed_ip=%s\n", ip.String()))
			}
		} else {
			request.WriteString(heredoc.Doc(`
				allowed_ip=0.0.0.0/0
				allowed_ip=::0/0
			`))
		}
	}

	setting := &DeviceSetting{ipcRequest: request.String(), dns: conf.DNS, deviceAddr: conf.Endpoint, mtu: conf.MTU}
	return setting, nil
}

// StartWireguard creates a tun interface on netstack given a configuration
func StartWireguard(conf *DeviceConfig, verbose bool, ctx context.Context) (*VirtualTun, error) {
	setting, err := createIPCRequest(conf)
	if err != nil {
		return nil, err
	}

	tun, tnet, err := netstack.CreateNetTUN(setting.deviceAddr, setting.dns, setting.mtu)
	if err != nil {
		return nil, err
	}

	logLevel := device.LogLevelVerbose
	if !verbose {
		logLevel = device.LogLevelSilent
	}

	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(logLevel, ""))
	err = dev.IpcSet(setting.ipcRequest)
	if err != nil {
		return nil, err
	}

	err = dev.Up()
	if err != nil {
		return nil, err
	}

	return &VirtualTun{
		Tnet:      tnet,
		SystemDNS: len(setting.dns) == 0,
		Verbose:   verbose,
		Logger: DefaultLogger{
			verbose: verbose,
		},
		Dev: dev,
		Ctx: ctx,
	}, nil
}
