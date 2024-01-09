# Localtime

A daemon for keeping the system timezone up-to-date based on the current location.

## Configuring GeoClue

Unfortunately, GeoClue is stubbornly a Gnome and desktop-centric service. There's no way (literally)
to use it from a system service without manual intervention, unless you can convince them to add
your app to a whitelist. You can:

1. Configure a geoclue agent (e.g., the demo agent), and configure the demo agent to allow
   `localtimed`. I've tried having localtime autostart this if it can't find an agent running, but
   GeoClue insisted that there was no agent running anyways.
2. Disable geoclue agents by clearing the agent whitelist in `/etc/geoclue/geoclue.conf`. This is by
   far the simplest approach, but it will allow arbitrary apps to get your location.

## Install

If possible, use your package manager:

* [Arch Linux (AUR)](https://aur.archlinux.org/packages/localtime-git)

Otherwise, follow the instructions below.

### Manual Install

    $ make
    $ sudo make install

## Enable and start

    $ sudo systemctl enable --now geoclue-demo-agent.service localtime.service

## Dependencies

### Runtime

* geoclue2
* systemd >= 235 (for full dynamic user support)
* dbus
* polkit (to run as a non-root user)

### Build

* go
* make
* m4
