# Localtime

A daemon for keeping the system timezone up-to-date based on the current location.

## Install

If possible, use your package manager:

* [Arch Linux (AUR)](https://aur.archlinux.org/packages/localtime-git)

Otherwise, follow the instructions below.

### Manual Install

    $ make
    $ sudo make install

## Enable and start

    $ sudo systemctl enable --now localtime.service

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
