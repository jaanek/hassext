package chromecast

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/zerodha/logf"
)

type Chromecasts interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	ChromecastByDeviceName(name string) Chromecast
}

type ccs struct {
	prefix      string
	log         logf.Logger
	done        chan struct{}
	doneOnce    sync.Once
	ifaceName   string
	deviceNames []string
	devices     []Chromecast
	cert        *tls.Certificate
}

func NewDevices(log logf.Logger, ifaceName string, deviceNames []string, cert *tls.Certificate) Chromecasts {
	return &ccs{
		prefix:      "[chromecasts] ",
		log:         log,
		done:        make(chan struct{}),
		ifaceName:   ifaceName,
		deviceNames: deviceNames,
		cert:        cert,
	}
}

func (c *ccs) Start(ctx context.Context) error {
	// If we need to look on a specific network interface for mdns or
	// for finding a network ip to host from, ensure that the network
	// interface exists.
	var iface *net.Interface
	if c.ifaceName != "" {
		var err error
		if iface, err = net.InterfaceByName(c.ifaceName); err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to find interface %q", c.ifaceName))
		}
	}

	// start polling loop or react to triggering event
	var syncInterval = time.Minute * 1
	ticker := time.NewTicker(syncInterval)
	var triggerCheckQueue = make(chan struct{})
	var tickerReset = make(chan struct{})
	defer ticker.Stop()
	go func() {
		triggerCheckQueue <- struct{}{}
	}()
	for {
		select {
		case <-triggerCheckQueue:
			c.log.Info(c.prefix + "(Re)connecting chromecasts ...")
			err := c.connectDevices(iface)
			if err != nil {
				c.log.Error(c.prefix+"error connecting devices", "error", err)
			}
			go func() {
				tickerReset <- struct{}{}
			}()
		case <-tickerReset:
			ticker.Stop()
			ticker = time.NewTicker(syncInterval)
		case <-ticker.C:
			go func() {
				triggerCheckQueue <- struct{}{}
			}()
		case <-c.done:
			c.log.Info(c.prefix + "Exiting chromecasts re-connection process")
			return nil
		}
	}
}

func (c *ccs) Stop(ctx context.Context) error {
	c.doneOnce.Do(func() {
		close(c.done)
	})
	var connectTimeout = 5 * time.Second
	for _, d := range c.devices {
		dctx, dcancel := context.WithTimeout(context.Background(), connectTimeout)
		defer dcancel()
		d.Disconnect(dctx)
	}
	return nil
}

func (c *ccs) ChromecastByDeviceName(name string) Chromecast {
	for _, cc := range c.devices {
		if cc.DNSEntry().GetName() == name {
			return cc
		}
	}
	return nil
}

func (c *ccs) connectDevices(iface *net.Interface) error {
	// get latest dns entries
	var entries []CastDNSEntry
	var err error
	var dnsTimeoutSeconds int = 5
	if entries, err = findCastDNSEntries(c.prefix, c.log, iface, dnsTimeoutSeconds, c.deviceNames); err != nil {
		return errors.Wrap(err, "error while searching for chromecast dns entries")
	}
	c.log.Info(c.prefix+"Found configured dns entries!", "entries", entries, "config device names", c.deviceNames)

	// map all device entries by device names
	var entriesMap = map[string]*CastDNSEntry{}
	for _, deviceName := range c.deviceNames {
		var foundEntry *CastDNSEntry
		for _, e := range entries {
			if e.GetName() == deviceName {
				foundEntry = &e
				break
			}
		}
		if foundEntry == nil {
			c.log.Warn(c.prefix+"Chromecast dns entry not found, cannot connect!", "deviceName", deviceName)
		} else {
			entriesMap[(*foundEntry).GetUUID()] = foundEntry
		}
	}

	// check if connected chromecast is responding, if not then remove if from the list
	var connectTimeout = 5 * time.Second
	var liveDevices []Chromecast
	for _, cc := range c.devices {
		resp, err := cc.GetReceiverStatus()
		if err != nil {
			c.log.Warn(c.prefix+"Chromecast receiver status check error", "name", cc.DNSEntry().GetName(), "error", err)
			dctx, dcancel := context.WithTimeout(context.Background(), connectTimeout)
			err = cc.Disconnect(dctx)
			if err != nil {
				c.log.Debug(c.prefix+"Chromecast disconnect error", "name", cc.DNSEntry().GetName(), "error", err)
			}
			dcancel()
			continue
		}
		liveDevices = append(liveDevices, cc)
		c.log.Info(c.prefix+"Chromecast receiver status check", "name", cc.DNSEntry().GetName(), "resp", resp)
	}
	c.devices = liveDevices

	// check if we need to re-connect
	for uuid, e := range entriesMap {
		var entry = *e
		// find existing connected device
		var d Chromecast
		for _, c := range c.devices {
			if uuid == c.DNSEntry().GetUUID() {
				d = c
				break
			}
		}
		if d != nil {
			var sameIpPort = entry.GetAddr() == d.DNSEntry().GetAddr() && entry.GetPort() == d.DNSEntry().GetPort()
			if sameIpPort {
				c.log.Info(c.prefix+"Not re-connecting to chromecast. Same IP & Port", "name", entry.GetName(), "host", entry.GetAddr(), "port", entry.GetPort())
				continue
			}
			// existing device ip or port are not same. disconnect & connect again
			dctx, dcancel := context.WithTimeout(context.Background(), connectTimeout)
			defer dcancel()
			if err = d.Disconnect(dctx); err != nil {
				c.log.Error(c.prefix+"IP or Port changed. Disconnect error to chromecast!", "error", err, "name", d.DNSEntry().GetName(), "host", d.DNSEntry().GetAddr(), "port", d.DNSEntry().GetPort())
			}
			// set new dns entry & connect again
			d.SetDNSEntry(entry)
			ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
			defer cancel()
			if err = d.Connect(ctx); err != nil {
				c.log.Error(c.prefix+"Cannot re-connect to chromecast!", "error", err, "name", d.DNSEntry().GetName(), "host", d.DNSEntry().GetAddr(), "port", d.DNSEntry().GetPort())
				continue
			}
			c.log.Info(c.prefix+"Successfully re-connected to chromecast!", "name", entry.GetName(), "host", entry.GetAddr(), "port", entry.GetPort())
		} else {
			var d = New(c.log, entry, c.cert)
			// connect new device
			ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
			defer cancel()
			if err = d.Connect(ctx); err != nil {
				c.log.Error(c.prefix+"Cannot connect to chromecast!", "error", err, "name", entry.GetName(), "host", entry.GetAddr(), "port", entry.GetPort())
				continue
			}
			c.devices = append(c.devices, d)
			c.log.Info(c.prefix+"Successfully connected to chromecast!", "name", entry.GetName(), "host", entry.GetAddr(), "port", entry.GetPort())
		}
	}
	return nil
}
