// Copyright (c) Steven Allen 2016
//
// This file is part of localtime.
//
// Localtime is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, version 3 only.
//
// Foobar is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with localtime.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/bradfitz/latlong"
	"github.com/godbus/dbus/v5"
	"golang.org/x/sys/unix"
)

const (
	GeoclueBus  = "org.freedesktop.GeoClue2"
	TimedateBus = "org.freedesktop.timedate1"

	GeoclueClientInterface   = GeoclueBus + ".Client"
	GeoclueLocationInterface = GeoclueBus + ".Location"
	GeoclueManagerInterface  = GeoclueBus + ".Manager"
	TimedateInterface        = TimedateBus

	GeoclueAgent = "/usr/lib/geoclue-2.0/demos/agent"
)

//nolint
const (
	GCLUE_ACCURACY_LEVEL_NONE         = 0
	GCLUE_ACCURACY_LEVEL_COUNTRY      = 1
	GCLUE_ACCURACY_LEVEL_CITY         = 4
	GCLUE_ACCURACY_LEVEL_NEIGHBORHOOD = 5
	GCLUE_ACCURACY_LEVEL_STREET       = 6
	GCLUE_ACCURACY_LEVEL_EXACT        = 8
)

func NewGeoclueClient(conn *dbus.Conn) (*GeoclueClient, error) {
	manager := conn.Object(
		GeoclueBus,
		"/org/freedesktop/GeoClue2/Manager",
	)

	var clientPath dbus.ObjectPath

	if err := manager.Call(
		GeoclueManagerInterface+".GetClient",
		0,
	).Store(&clientPath); err != nil {
		return nil, err
	}

	clientObject := conn.Object(
		GeoclueBus,
		clientPath,
	)

	if err := clientObject.SetProperty(
		GeoclueClientInterface+".DistanceThreshold",
		dbus.MakeVariant(uint32(1000)),
	); err != nil {
		return nil, err
	}

	if err := clientObject.SetProperty(
		GeoclueClientInterface+".RequestedAccuracyLevel",
		dbus.MakeVariant(uint32(GCLUE_ACCURACY_LEVEL_CITY)),
	); err != nil {
		return nil, err
	}

	if err := clientObject.SetProperty(
		GeoclueClientInterface+".DesktopId",
		dbus.MakeVariant("localtimed"),
	); err != nil {
		return nil, err
	}

	return &GeoclueClient{
		client: clientObject,
		conn:   conn,
		done:   make(chan struct{}),
	}, nil
}

func setTimezone(conn *dbus.Conn, timezone string) error {
	timedate := conn.Object(TimedateBus, "/org/freedesktop/timedate1")
	return timedate.Call(
		TimedateInterface+".SetTimezone",
		0,
		timezone,
		false).Err
}

type Location struct {
	Latitude  float64
	Longitude float64
}

type GeoclueClient struct {
	client dbus.BusObject
	conn   *dbus.Conn
	done   chan struct{}
}

func (g *GeoclueClient) Close() error {
	err := g.client.Call(
		GeoclueClientInterface+".Stop",
		0,
	).Err

	close(g.done)
	return err
}

func (g *GeoclueClient) Start() (chan Location, error) {

	// We don't need to add any matches because the signals are directed at us.

	signals := make(chan *dbus.Signal, 10)
	g.conn.Signal(signals)

	stopAgent := func() {}

	// Try once without starting an agent.
	if call := g.client.Call(
		GeoclueClientInterface+".Start",
		0,
	); call.Err != nil {
		if !strings.Contains(call.Err.Error(), "no agent for UID") {
			return nil, call.Err
		}

		// Now try to start the demo agent.
		agentCmd := exec.Command(GeoclueAgent)
		if err := agentCmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start GeoClue2 agent: %w", err)
		}

		stopAgent = func() {
			agentCmd.Process.Signal(unix.SIGTERM)
			agentCmd.Wait()
		}
		if call := g.client.Call(
			GeoclueClientInterface+".Start",
			0,
		); call.Err != nil {
			stopAgent()
			return nil, call.Err
		}
	}

	output := make(chan Location)
	go func() {
		for {
			select {
			case signal := <-signals:
				var (
					oldLocation dbus.ObjectPath
					newLocation dbus.ObjectPath
				)

				if signal.Name != GeoclueClientInterface+".LocationUpdated" {
					continue
				}

				if err := dbus.Store(
					signal.Body,
					&oldLocation,
					&newLocation,
				); err != nil {
					panic("signal type")
				}

				output <- g.readLocation(newLocation)
			case <-g.done:
				stopAgent()
				close(output)
				return
			}
		}
	}()

	return output, nil
}

func (g *GeoclueClient) readLocation(location dbus.ObjectPath) Location {
	locationObject := g.conn.Object(GeoclueBus, location)

	lat, err := locationObject.GetProperty(
		GeoclueLocationInterface + ".Latitude",
	)

	if err != nil {
		panic("Missing latitude")
	}

	lon, err := locationObject.GetProperty(
		GeoclueLocationInterface + ".Longitude",
	)

	if err != nil {
		panic("Missing longitude")
	}

	return Location{
		Latitude:  lat.Value().(float64),
		Longitude: lon.Value().(float64),
	}
}

func main() {
	conn, err := dbus.SystemBus()

	if err != nil {
		log.Fatalln("Failed to connect to system bus:", err)
	}

	client, err := NewGeoclueClient(conn)
	if err != nil {
		log.Fatalln("Failed to initialize GeoClue2 Client:", err)
	}

	updates, err := client.Start()
	if err != nil {
		log.Fatalln("Failed to start GeoClue2 Client:", err)
	}
	defer client.Close()

	old_timezone := ""
	for loc := range updates {
		if err != nil {
			log.Println("Failed to get location:", err)
			continue
		}

		new_timezone := latlong.LookupZoneName(loc.Latitude, loc.Longitude)
		if new_timezone == "" {
			log.Println("Failed to get timezone from location:", loc)
			continue
		}
		if new_timezone == old_timezone {
			continue
		}
		if err := setTimezone(conn, new_timezone); err != nil {
			log.Println("Failed to set timezone:", new_timezone)
			continue
		}
		log.Println("Updated timezone to:", new_timezone)
		old_timezone = new_timezone
	}
}
