package main

import (
	"errors"
	"fmt"
	"github.com/bradfitz/latlong"
	"github.com/guelfey/go.dbus"
	"os"
)

func NewGeoclueClient(conn *dbus.Conn) (*GeoclueClient, error) {
	manager := conn.Object(
		"org.freedesktop.GeoClue2",
		dbus.ObjectPath("/org/freedesktop/GeoClue2/Manager"))

	var clientPath dbus.ObjectPath

	if err := manager.Call("org.freedesktop.GeoClue2.Manager.GetClient", 0).Store(&clientPath); err != nil {
		return nil, err
	}

	clientObject := conn.Object(
		"org.freedesktop.GeoClue2",
		clientPath,
	)

	if call := clientObject.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"org.freedesktop.GeoClue2.Client",
		"DistanceThreshold",
		dbus.MakeVariant(uint32(1000))); call.Err != nil {
		return nil, call.Err
	}

	if call := clientObject.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"org.freedesktop.GeoClue2.Client",
		"DesktopId",
		dbus.MakeVariant("tzupdated")); call.Err != nil {
		return nil, call.Err
	}

	return &GeoclueClient{
		client: clientObject,
		conn:   conn,
		done:   nil,
	}, nil
}

func setTimezone(conn *dbus.Conn, timezone string) error {
	timedate := conn.Object("org.freedesktop.timedate1", "/org/freedesktop/timedate1")
	return timedate.Call(
		"org.freedesktop.timedate1.SetTimezone",
		0,
		timezone,
		false).Err
}

type Location struct {
	Latitude  float64
	Longitude float64
}

type GeoclueClient struct {
	client *dbus.Object
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
		"org.freedesktop.GeoClue2.Client.Stop",
		0,
	).Err
}

func (self *GeoclueClient) Start() (chan Location, error) {

	if self.done != nil {
		return nil, errors.New("Already started")
	}

	self.done = make(chan bool)

	filter := fmt.Sprintf("type='%s',sender='%s',interface='%s',member='%s',path='%s'",
		"signal",
		"org.freedesktop.GeoClue2",
		"dbus.freedesktop.GeoClue2.Client",
		"LocationUpdated",
		self.client.Path(),
	)

	if call := self.conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").Call("org.freedesktop.DBus.AddMatch", dbus.FlagNoAutoStart, filter); call.Err != nil {
		return nil, call.Err
	}

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

				if signal.Name != "org.freedesktop.GeoClue2.Client.LocationUpdated" {
					continue
				}

				if err := dbus.Store(signal.Body, &oldLocation, &newLocation); err != nil {
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
		"org.freedesktop.GeoClue2.Client.Start",
		0,
	); call.Err != nil {
		self.done <- true
		self.done = nil
		return nil, call.Err
	}
	return output, nil
}

func (self *GeoclueClient) readLocation(location dbus.ObjectPath) (loc Location) {
	locationObject := self.conn.Object("org.freedesktop.GeoClue2", location)

	lat, err := locationObject.GetProperty(
		"org.freedesktop.GeoClue2.Location.Latitude",
	)

	if err != nil {
		panic("Missing latitude")
	}

	loc.Latitude = lat.Value().(float64)

	lon, err := locationObject.GetProperty(
		"org.freedesktop.GeoClue2.Location.Longitude",
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
		fmt.Fprintln(os.Stderr, "Failed to connect to system bus:", err)
		os.Exit(1)
	}

	client, err := NewGeoclueClient(conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize GeoClue2 Client:", err)
		os.Exit(1)
	}

	updates, err := client.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start GeoClue2 Client:", err)
		os.Exit(1)
	}
	defer client.Stop()

	old_timezone := ""
	for loc := range updates {
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get location:", err)
			continue
		}

		new_timezone := latlong.LookupZoneName(loc.Latitude, loc.Longitude)
		if new_timezone == "" {
			fmt.Fprintln(os.Stderr, "Failed to get timezone from location:", loc)
			continue
		}
		if new_timezone == old_timezone {
			continue
		}
		if err := setTimezone(conn, new_timezone); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to set timezone:", new_timezone)
			continue
		}
		old_timezone = new_timezone
	}
}
