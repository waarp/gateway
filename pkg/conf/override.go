package conf

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/config"
)

// LocalOverrides is a global instance of Override containing the local
// configuration overrides defined for this particular gateway node.
var LocalOverrides Override

// Override is a struct defining a list of settings local to a gateway instance
// (or node) which can be used to override settings defined at the cluster level.
type Override struct {
	mux             sync.RWMutex
	filename        string
	NodeIdentifier  string          `no-ini:"true"`
	ListenAddresses AddressOverride `group:"Address Indirection"`
}

// InitOverride initialises the LocalOverrides global instance with a new,
//correctly initialised, instance of Override.
func InitOverride(filename string) {
	LocalOverrides.mux.Lock()
	defer LocalOverrides.mux.Unlock()
	LocalOverrides.filename = filename
	LocalOverrides.ListenAddresses.over = &LocalOverrides
	LocalOverrides.ListenAddresses.init()
}

type overrideWrite struct {
	ListenAddresses addressOverrideWrite `group:"Address Indirection"`
}

func (o *Override) writeFile() error {
	file, err := os.OpenFile(o.filename, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	o.writeTo(file)
	return file.Close()
}

// WriteFile writes the content of Override to the given .ini file.
func (o *Override) WriteFile() error {
	o.mux.RLock()
	defer o.mux.RUnlock()
	return o.writeFile()
}

func (o *Override) makeWrite() *overrideWrite {
	return &overrideWrite{
		ListenAddresses: o.ListenAddresses.makeWrite(),
	}
}

func (o *Override) writeTo(w io.Writer) {
	overW := o.makeWrite()
	parser := config.NewParser(overW)
	parser.Write(w)
}

// AddressOverride is a struct defining a local list of address indirections
// (or overrides) which allows a gateway instance to replace an address defined
// at the cluster level with another one. Address indirections are given using
// the `IndirectAddress` ini option, followed by the target address and the real
// address separated by a '->'. The option name must be repeated for each indirection.
// This means that the indirections should have the following format:
//
//    IndirectAddress = localhost -> 127.0.0.1
//    IndirectAddress = example.com -> 8.8.8.8:80
//    IndirectAddress = 192.168.1.1 -> [::1]:8080
//
// Do note that, although not mandatory, IPv6 addresses should be surrounded with
// [brackets] to avoid confusion when adding a port number at the end.
type AddressOverride struct {
	over       *Override
	addressMap map[string]string
	Listen     func(string) error `ini-name:"IndirectAddress" description:"Replace the target address with another one"`
}

type addressOverrideWrite struct {
	Listen []string `ini-name:"IndirectAddress" description:"Replace the target address with another one"`
}

func (a *AddressOverride) init() {
	a.Listen = a.parseListen
	if a.addressMap == nil {
		a.addressMap = map[string]string{}
	}
}

func (a *AddressOverride) parseListen(val string) error {
	slice := strings.Split(val, "->")
	if len(slice) < 2 {
		return fmt.Errorf("malformed address indirection '%s' (missing '->' separator)", val)
	} else if len(slice) > 2 {
		return fmt.Errorf("malformed address indirection '%s' (too many '->' separators)", val)
	}
	target := strings.TrimSpace(slice[0])
	addr := strings.TrimSpace(slice[1])
	if _, ok := a.addressMap[target]; ok {
		return fmt.Errorf("duplicate address indirection target '%s'", target)
	}
	a.addressMap[target] = addr
	return nil
}

// GetIndirection returns the real address associated with the given target address.
func (a *AddressOverride) GetIndirection(target string) string {
	a.over.mux.RLock()
	defer a.over.mux.RUnlock()
	return a.addressMap[target]
}

// GetRealAddress returns the real address obtained after indirection has been
// applied to the given address. The address must have both a host & a port.
// If no indirection exist for the given address or for its host, the address is
// returned unchanged (because it already is a real address).
func (a *AddressOverride) GetRealAddress(target string) (string, error) {
	a.over.mux.RLock()
	defer a.over.mux.RUnlock()

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return "", err
	}
	if realAddr := a.addressMap[target]; realAddr != "" {
		return realAddr, nil
	}
	if realHost := a.addressMap[host]; realHost != "" {
		return net.JoinHostPort(realHost, port), nil
	}
	return target, nil
}

// GetAllIndirections return a map containing all the address indirections present
// in the configuration. The returned map is a copy of the real, global map.
func (a *AddressOverride) GetAllIndirections() map[string]string {
	a.over.mux.RLock()
	defer a.over.mux.RUnlock()
	newMap := map[string]string{}
	for k, v := range a.addressMap {
		newMap[k] = v
	}
	return newMap
}

// AddIndirection adds the given address indirection to the local overrides. The
// associated file will also be updated. If the indirection already exist, the
// old value will be overwritten.
func (a *AddressOverride) AddIndirection(target, real string) error {
	a.over.mux.Lock()
	defer a.over.mux.Unlock()
	a.addressMap[target] = real

	return a.over.writeFile()
}

// RemoveIndirection removes the given address indirection from the local overrides,
// and from the associated override file.
func (a *AddressOverride) RemoveIndirection(target string) error {
	a.over.mux.Lock()
	defer a.over.mux.Unlock()
	delete(a.addressMap, target)

	return a.over.writeFile()
}

func (a *AddressOverride) makeWrite() addressOverrideWrite {
	slice := make([]string, len(a.addressMap))
	var i int
	for target, addr := range a.addressMap {
		slice[i] = target + " -> " + addr
		i++
	}
	return addressOverrideWrite{Listen: slice}
}
