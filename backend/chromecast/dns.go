package chromecast

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/grandcat/zeroconf"
	"github.com/jaanek/hassext/bytes"
	"github.com/zerodha/logf"
)

func findCastDNSEntries(prefix string, log logf.Logger, iface *net.Interface, dnsTimeoutSeconds int, deviceNames []string) ([]CastDNSEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(dnsTimeoutSeconds))
	defer cancel()
	castEntryChan, err := DiscoverCastDNSEntries(prefix, log, ctx, iface)
	if err != nil {
		return nil, err
	}

	allEntries := []CastEntry{}
	foundEntries := []CastDNSEntry{}
	for entry := range castEntryChan {
		if bytes.ContainsString(deviceNames, entry.DeviceName) {
			foundEntries = append(foundEntries, entry)
		}
		allEntries = append(allEntries, entry)
	}

	// Always return entries in deterministic order.
	sort.Slice(foundEntries, func(i, j int) bool { return foundEntries[i].GetName() < foundEntries[j].GetName() })

	// debug
	for i, d := range allEntries {
		log.Info(prefix+"Found dns entry", "#", i+1, "device", d.Device, "deviceName", d.DeviceName, "addr", d.AddrV4, "port", d.Port, "uuid", d.UUID)
	}
	for _, d := range foundEntries {
		log.Info(prefix+"Found config dns entries", "deviceName", d.GetName(), "addr", d.GetAddr(), "port", d.GetPort(), "uuid", d.GetUUID())
	}
	return foundEntries, nil
}

// CastDNSEntry is the interface that satisfies a Cast type.
type CastDNSEntry interface {
	GetName() string
	GetUUID() string
	GetAddr() string
	GetPort() int
}

// CastEntry is the concrete cast entry type.
type CastEntry struct {
	AddrV4 net.IP
	AddrV6 net.IP
	Port   int

	Name string
	Host string

	UUID       string
	Device     string
	Status     string
	DeviceName string
	InfoFields map[string]string
}

// GetUUID returns a unqiue id of a cast entry.
func (e CastEntry) GetUUID() string {
	return e.UUID
}

// GetName returns the identified name of a cast entry.
func (e CastEntry) GetName() string {
	return e.DeviceName
}

// GetAddr returns the IPV4 of a cast entry if it is not nil otherwise the IPV6.
func (e CastEntry) GetAddr() string {
	if e.AddrV4 != nil {
		return e.AddrV4.String()
	} else {
		return fmt.Sprintf("[%s]", e.AddrV6.String())
	}
}

// GetPort returns the port of a cast entry.
func (e CastEntry) GetPort() int {
	return e.Port
}

// DiscoverCastDNSEntryByName returns the first cast dns device
// found that matches the name.
func DiscoverCastDNSEntryByName(prefix string, log logf.Logger, ctx context.Context, iface *net.Interface, name string) (CastEntry, error) {
	castEntryChan, err := DiscoverCastDNSEntries(prefix, log, ctx, iface)
	if err != nil {
		return CastEntry{}, err
	}

	for d := range castEntryChan {
		if d.DeviceName == name {
			return d, nil
		}
	}
	return CastEntry{}, fmt.Errorf("No cast device found with name %q", name)
}

// DiscoverCastDNSEntries will return a channel with any cast dns entries
// found.
func DiscoverCastDNSEntries(prefix string, log logf.Logger, ctx context.Context, iface *net.Interface) (<-chan CastEntry, error) {
	var opts = []zeroconf.ClientOption{}
	if iface != nil {
		opts = append(opts, zeroconf.SelectIfaces([]net.Interface{*iface}))
	}
	resolver, err := zeroconf.NewResolver(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create new zeroconf resolver: %w", err)
	}

	castDNSEntriesChan := make(chan CastEntry, 5)
	entriesChan := make(chan *zeroconf.ServiceEntry, 5)
	go func() {
		if err := resolver.Browse(ctx, "_googlecast._tcp", "local", entriesChan); err != nil {
			log.Error(prefix+"unable to browser for mdns entries", err)
			return
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(castDNSEntriesChan)
				return
			case entry := <-entriesChan:
				if entry == nil {
					continue
				}
				castEntry := CastEntry{
					Port: entry.Port,
					Host: entry.HostName,
				}
				if len(entry.AddrIPv4) > 0 {
					castEntry.AddrV4 = entry.AddrIPv4[0]
				}
				if len(entry.AddrIPv6) > 0 {
					castEntry.AddrV6 = entry.AddrIPv6[0]
				}
				infoFields := make(map[string]string, len(entry.Text))
				for _, value := range entry.Text {
					if kv := strings.SplitN(value, "=", 2); len(kv) == 2 {
						key := kv[0]
						val := kv[1]

						infoFields[key] = val

						switch key {
						case "fn":
							castEntry.DeviceName = decode(val)
						case "md":
							castEntry.Device = decode(val)
						case "id":
							castEntry.UUID = val
						}
					}
				}
				castEntry.InfoFields = infoFields
				castDNSEntriesChan <- castEntry
			}
		}
	}()
	return castDNSEntriesChan, nil
}

// decode attempts to decode the passed in string using escaped utf8 bytes.
// some DNS entries for other languages seem to include utf8 escape sequences as
// part of the name.
func decode(val string) string {
	if strings.Index(val, "\\") == -1 {
		return val
	}

	var (
		r        []rune
		toDecode []byte
	)

	decodeRunes := func() {
		if len(toDecode) > 0 {
			for len(toDecode) > 0 {
				rr, size := utf8.DecodeRune(toDecode)
				r = append(r, rr)
				toDecode = toDecode[size:]
			}
			toDecode = []byte{}
		}
	}

	for i := 0; i < len(val); {
		if val[i] == '\\' {
			if i+3 < len(val) {
				v, err := strconv.Atoi(val[i+1 : i+4])
				if err == nil {
					toDecode = append(toDecode, byte(v))
					i += 4
					continue
				}
			}
		}
		decodeRunes()
		r = append(r, rune(val[i]))
		i++
	}
	decodeRunes()
	return string(r)
}
