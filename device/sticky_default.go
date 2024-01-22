//go:build !linux

package device

import (
	"github.com/uoosef/wireguard-go/conn"
	"github.com/uoosef/wireguard-go/rwcancel"
)

func (device *Device) startRouteListener(bind conn.Bind) (*rwcancel.RWCancel, error) {
	return nil, nil
}
