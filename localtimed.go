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
	"errors"
	"log"

	"github.com/bradfitz/latlong"
	"github.com/godbus/dbus/v5"
)

const (
	GeoclueBus  = "org.freedesktop.GeoClue2"
	TimedateBus = "org.freedesktop.timedate1"

	GeoclueClientInterface   = GeoclueBus + ".Client"
	GeoclueLocationInterface = GeoclueBus + ".Location"
	GeoclueManagerInterface  = GeoclueBus + ".Manager"
	TimedateInterface        = TimedateBus
)

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
		done:   nil,
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
	done   chan bool
}

func (self *GeoclueClient) Stop() error {
	if self.done != nil {
		return errors.New("Not Running")
	}

	self.done <- true
	self.done = nil

	return self.client.Call(
		GeoclueClientInterface+".Stop",
		0,
	).Err
}

func (self *GeoclueClient) Start() (chan Location, error) {

	if self.done != nil {
		return nil, errors.New("Already started")
	}

	self.done = make(chan bool)

	// We don't need to add any matches because the signals are directed at us.

	signals := make(chan *dbus.Signal, 10)
	self.conn.Signal(signals)

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

				output <- self.readLocation(newLocation)
			case <-self.done:
				close(output)
				return
			}
		}
	}()

	if call := self.client.Call(
		GeoclueClientInterface+".Start",
		0,
	); call.Err != nil {
		self.done <- true
		self.done = nil
		return nil, call.Err
	}
	return output, nil
}

func (self *GeoclueClient) readLocation(location dbus.ObjectPath) (loc Location) {
	locationObject := self.conn.Object(GeoclueBus, location)

	lat, err := locationObject.GetProperty(
		GeoclueLocationInterface + ".Latitude",
	)

	if err != nil {
		panic("Missing latitude")
	}

	loc.Latitude = lat.Value().(float64)

	lon, err := locationObject.GetProperty(
		GeoclueLocationInterface + ".Longitude",
	)

	if err != nil {
		panic("Missing longitude")
	}
	loc.Longitude = lon.Value().(float64)
	return
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
	defer client.Stop()

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
		old_timezone = new_timezone
	}
}
