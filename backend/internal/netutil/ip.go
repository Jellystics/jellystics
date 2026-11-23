// Package netutil holds the IP classification shared by the tracker (local vs
// remote plays) and the geolocation task.
package netutil

import (
	"net/netip"
	"strings"
)

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// Parse reads an IP as found in Jellyfin's RemoteEndPoint: bare, "ip:port",
// "[v6]:port" or IPv4-mapped IPv6. Zones are dropped.
func Parse(s string) (netip.Addr, bool) {
	s = strings.TrimSpace(s)
	var addr netip.Addr
	if ap, err := netip.ParseAddrPort(s); err == nil {
		addr = ap.Addr()
	} else if a, err := netip.ParseAddr(strings.Trim(s, "[]")); err == nil {
		addr = a
	} else {
		return netip.Addr{}, false
	}
	return addr.Unmap().WithZone(""), true
}

// IsLocal reports addresses that are never reachable from the internet:
// RFC1918, ULA (fc00::/7), loopback, link-local, multicast, unspecified.
func IsLocal(a netip.Addr) bool {
	return a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() ||
		a.IsLinkLocalMulticast() || a.IsMulticast() || a.IsUnspecified()
}

// IsCGNAT reports the shared address space 100.64.0.0/10 (carrier-grade NAT,
// Tailscale): treated as remote for plays but not geolocatable.
func IsCGNAT(a netip.Addr) bool { return cgnat.Contains(a) }

// IsGeolocatable reports public addresses worth a geolocation lookup.
func IsGeolocatable(a netip.Addr) bool { return !IsLocal(a) && !IsCGNAT(a) }
