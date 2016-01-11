#!/usr/bin/env python

import dbus

from gi.repository import GObject
from dbus.mainloop.glib import DBusGMainLoop

from tzwhere.tzwhere import tzwhere

PROPERTIES_INTERFACE_NAME = 'org.freedesktop.DBus.Properties'

GEOCLUE2_BUS_NAME = 'org.freedesktop.GeoClue2'
GEOCLUE_MANAGER_INTERFACE_NAME = GEOCLUE2_BUS_NAME + '.Manager'
GEOCLUE_CLIENT_INTERFACE_NAME = GEOCLUE2_BUS_NAME + '.Client'
GEOCLUELOCATION_INTERFACE_NAME = GEOCLUE2_BUS_NAME + '.Location'

TIMEDATE_BUS_NAME = 'org.freedesktop.timedate1'
TIMEDATE_INTERFACE_NAME = 'org.freedesktop.timedate1'

def get_geoclue(bus):
    manager = dbus.Interface(
        bus.get_object(GEOCLUE2_BUS_NAME,
                       '/org/freedesktop/GeoClue2/Manager'),
        GEOCLUE_MANAGER_INTERFACE_NAME)

    client_object = bus.get_object(GEOCLUE2_BUS_NAME, manager.GetClient())

    client_properties = dbus.Interface(client_object, PROPERTIES_INTERFACE_NAME)
    client_properties.Set(GEOCLUE_CLIENT_INTERFACE_NAME, "DistanceThreshold", dbus.UInt32(1000))
    client_properties.Set(GEOCLUE_CLIENT_INTERFACE_NAME, "DesktopId", dbus.String("tzupdater"))

    return dbus.Interface(client_object, GEOCLUE_CLIENT_INTERFACE_NAME)

def get_timedate(bus):
    return dbus.Interface(bus.get_object(TIMEDATE_BUS_NAME,
                                  '/org/freedesktop/timedate1'),
                   TIMEDATE_INTERFACE_NAME)

class TZUpdater:
    def __init__(self, bus):
        self.bus = bus
        self.timezone = None
        self.geoclue = get_geoclue(bus)
        self.tzwhere = tzwhere(shapely=True, forceTZ=True)
        self.geoclue.connect_to_signal('LocationUpdated', lambda old, new: self._location_updated(new))

    def __enter__(self):
        self.geoclue.Start()

    def __exit__(self, _type, _value, _traceback):
        self.geoclue.Stop()

    def _location_updated(self, loc):
        print("getting location")
        location = dbus.Interface(self.bus.get_object(GEOCLUE2_BUS_NAME, loc), PROPERTIES_INTERFACE_NAME)

        latitude = location.Get(LOCATION_INTERFACE_NAME, 'Latitude')
        longitude = location.Get(LOCATION_INTERFACE_NAME, 'Longitude')
        
        print("getting timezone", latitude, longitude)
        new_timezone = self.tzwhere.tzNameAt(latitude, longitude)
        print("got timezone", new_timezone)
        if new_timezone == self.timezone:
            return

        self.timezone = new_timezone

        try:
            get_timedate(bus).SetTimezone(dbus.String(self.timezone), dbus.Boolean(False))
        except:
            print("Failed to set timezone")

def main():
    # In order to make asynchronous calls, we need to setup an event loop
    # Go ahead and try removing this, the 'connect_to_signal' method
    # at the bottom won't work
    dbus_loop = DBusGMainLoop(set_as_default = True)

    # We connect to the system bus as GeoClue2 is located there
    bus = dbus.SystemBus(mainloop = dbus_loop)

    with TZUpdater(bus):
        loop = GObject.MainLoop()
        loop.run()

if __name__ == "__main__":
    main()
