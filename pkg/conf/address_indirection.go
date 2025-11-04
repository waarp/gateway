package conf

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var (
	ErrNoOverrideFile      = errors.New("no override file configured")
	ErrIndirectionWithPort = errors.New("an address with port must redirect to another address with port")
	ErrIndirectionNoPort   = errors.New("an address without port must redirect to another address without port")
)

// addressOverride is a struct defining a local list of address indirections
// (or overrides) which allows a gateway instance to replace an address defined
// at the cluster level with another one. Address indirections are given using
// the `IndirectAddress` ini option, followed by the target address and the real
// address separated by a '->'. The option name must be repeated for each indirection.
// This means that the indirections should have the following format:
//
//	IndirectAddress = localhost -> 127.0.0.1
//	IndirectAddress = example.com -> 8.8.8.8:80
//	IndirectAddress = 192.168.1.1 -> [::1]:8080
//
// Do note that, although not mandatory, IPv6 addresses should be surrounded with
// [brackets] to avoid confusion when adding a port number at the end.
type addressOverride struct {
	addressMap   map[string]string
	Indirections []string `ini-name:"IndirectAddress" description:"Replace the target address with another one"`
}

func (a *addressOverride) parse() error {
	a.addressMap = map[string]string{}
	for _, val := range a.Indirections {
		slice := strings.Split(val, "->")
		if len(slice) < 2 { //nolint:mnd //too specific
			//nolint:err113 //this is a base error
			return fmt.Errorf("malformed address indirection '%s' (missing '->' separator)", val)
		} else if len(slice) > 2 { //nolint:mnd //too specific
			//nolint:err113 //this is a base error
			return fmt.Errorf("malformed address indirection '%s' (too many '->' separators)", val)
		}

		target := strings.TrimSpace(slice[0])
		addr := strings.TrimSpace(slice[1])

		if _, ok := a.addressMap[target]; ok {
			//nolint:err113 //this is a base error
			return fmt.Errorf("duplicate address indirection target '%s'", target)
		}

		if err := checkIndirectionConsistency(target, addr); err != nil {
			return err
		}

		a.addressMap[target] = addr
	}

	return nil
}

func checkIndirectionConsistency(target, redirect string) error {
	const missingPort = "missing port in address"

	targetHasPort := true
	redirectHasPort := true

	if _, _, err := net.SplitHostPort(target); err != nil {
		var addrErr *net.AddrError
		if errors.As(err, &addrErr) && addrErr.Err == missingPort {
			targetHasPort = false
		} else {
			return fmt.Errorf("failed to parse the target address %q: %w", target, err)
		}
	}

	if _, _, err := net.SplitHostPort(redirect); err != nil {
		var addrErr *net.AddrError
		if errors.As(err, &addrErr) && addrErr.Err == missingPort {
			redirectHasPort = false
		} else {
			return fmt.Errorf("failed to parse the real address %q: %w", redirect, err)
		}
	}

	if targetHasPort != redirectHasPort {
		if targetHasPort {
			return fmt.Errorf("address %q is missing a port number: %w",
				redirect, ErrIndirectionWithPort)
		}

		return fmt.Errorf("address %q should not have a port number: %w",
			redirect, ErrIndirectionNoPort)
	}

	return nil
}

func (a *addressOverride) update() {
	newIndirections := make([]string, len(a.addressMap))
	i := 0

	for target, redirect := range a.addressMap {
		newIndirections[i] = fmt.Sprintf("%s -> %s", target, redirect)
		i++
	}

	a.Indirections = newIndirections
}

// GetIndirection returns the real address associated with the given target
// address if it exists in the global LocalOverrides instance. Otherwise, it
// returns an empty string.
func GetIndirection(target string) string {
	if LocalOverrides == nil {
		return ""
	}

	LocalOverrides.overrideLock.RLock()
	defer LocalOverrides.overrideLock.RUnlock()

	return LocalOverrides.ListenAddresses.addressMap[target]
}

// GetRealAddress returns the real address obtained after indirection has been
// applied to the given address. The given address MUST have both a host & a port,
// otherwise the function will return an error. The returned address will always
// be a complete address with both a host & a port number.
//
// First, the function searches for an exact match (including the port number)
// of the given address. If a match is found, the associated "real" address is
// returned.
//
// If no exact match is found, the function will then search for a match of just
// the host part of the address (without the port). If a match is found, the
// returned address will be the concatenation of the new host with the old port.
//
// Finally, if no match is found for the host either, this means that the given
// address has no known indirection, and so the address is returned as is.
func GetRealAddress(host, port string) string {
	target := net.JoinHostPort(host, port)

	if LocalOverrides == nil {
		return target
	}

	LocalOverrides.overrideLock.RLock()
	defer LocalOverrides.overrideLock.RUnlock()

	if realAddr := LocalOverrides.ListenAddresses.addressMap[target]; realAddr != "" {
		return realAddr
	}

	if realHost := LocalOverrides.ListenAddresses.addressMap[host]; realHost != "" {
		return net.JoinHostPort(realHost, port)
	}

	return target
}

// GetAllIndirections return a map containing all the address indirections present
// in the configuration. The returned map is a copy of the real, global map.
func GetAllIndirections() map[string]string {
	if LocalOverrides == nil {
		return nil
	}

	LocalOverrides.overrideLock.RLock()
	defer LocalOverrides.overrideLock.RUnlock()

	newMap := map[string]string{}
	for k, v := range LocalOverrides.ListenAddresses.addressMap {
		newMap[k] = v
	}

	return newMap
}

// AddIndirection adds the given address indirection to the global LocalOverrides
// instance. The associated file will also be updated. If the indirection already
// exist, the old value will be overwritten.
func AddIndirection(targetAddr, realAddr string) error {
	if LocalOverrides == nil {
		return nil
	}

	LocalOverrides.overrideLock.Lock()
	defer LocalOverrides.overrideLock.Unlock()

	if err := checkIndirectionConsistency(targetAddr, realAddr); err != nil {
		return err
	}

	LocalOverrides.ListenAddresses.addressMap[targetAddr] = realAddr
	LocalOverrides.ListenAddresses.update()

	return LocalOverrides.writeFile()
}

// RemoveIndirection removes the given address indirection from the global
// LocalOverrides instance, and from its associated file.
func RemoveIndirection(target string) error {
	if LocalOverrides == nil {
		return nil
	}

	LocalOverrides.overrideLock.Lock()
	defer LocalOverrides.overrideLock.Unlock()

	delete(LocalOverrides.ListenAddresses.addressMap, target)
	LocalOverrides.ListenAddresses.update()

	return LocalOverrides.writeFile()
}
