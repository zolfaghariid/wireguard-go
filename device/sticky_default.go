//go:build !linux

package device

import (
	"github.com/bepass-org/wireguard-go/conn"
	"github.com/bepass-org/wireguard-go/rwcancel"
)

func (device *Device) startRouteListener(bind conn.Bind) (*rwcancel.RWCancel, error) {
	return nil, nil
}
