package conf

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/config"
)

var overrideLock sync.RWMutex

// Override is a struct defining a list of settings local to a gateway instance
// (or node) which can be used to override settings defined at the cluster level.
type Override struct {
	filename        string
	NodeIdentifier  string          `no-ini:"true"`
	ListenAddresses AddressOverride `group:"Address Indirection"`
}

// NewOverride returns a new, correctly initialised, instance of Override.
func NewOverride(filename string) Override {
	override := Override{
		filename: filename,
		ListenAddresses: AddressOverride{
			addressMap: map[string]string{},
		},
	}
	override.ListenAddresses.Listen = override.ListenAddresses.parseListen
	return override
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
	overrideLock.RLock()
	defer overrideLock.RUnlock()
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
// at the cluster level with another one.
type AddressOverride struct {
	addressMap map[string]string
	Listen     func(string) error `ini-name:"IndirectAddress" description:"Replace the target address with another one"`
}

type addressOverrideWrite struct {
	Listen []string `ini-name:"IndirectAddress" description:"Replace the target address with another one"`
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

// GetRealAddress returns the real address associated with the given target address.
func (a *AddressOverride) GetRealAddress(target string) string {
	overrideLock.RLock()
	defer overrideLock.RUnlock()
	return a.addressMap[target]
}

// GetAllIndirections return a map containing all the address indirections present
// in the configuration. The returned map is a copy of the real, global map.
func (a *AddressOverride) GetAllIndirections() map[string]string {
	overrideLock.RLock()
	defer overrideLock.RUnlock()
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
	overrideLock.Lock()
	defer overrideLock.Unlock()
	a.addressMap[target] = real

	return GlobalConfig.LocalOverrides.writeFile()
}

// RemoveIndirection removes the given address indirection from the local overrides,
// and from the associated override file.
func (a *AddressOverride) RemoveIndirection(target string) error {
	overrideLock.Lock()
	defer overrideLock.Unlock()
	delete(a.addressMap, target)

	return GlobalConfig.LocalOverrides.writeFile()
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
