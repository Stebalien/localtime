# Localtime

A daemon for keeping the system timezone up-to-date based on the current location.

## Install

If possible, use your package manager:

* [Arch Linux (AUR)](https://aur.archlinux.org/packages/localtime-git)

Otherwise, follow the instructions below.

### Manual Install

    $ make
    $ sudo make install

#### User and Group

If you have systemd-sysusers, either reboot or run manually run systemd-sysusers to create the localtimed user and group. If you don't have systemd-sysusers, you'll have to manually create the user and group:

    $ ### Only run this if you don't have systemd-sysusers. ###
    $ sudo make install-user

## Enable and start

    $ sudo systemctl enable --now localtime.service

## Dependencies

### Runtime

* geoclue2
* systemd
* dbus
* polkit (to run as a non-root user)

### Build

* go
* make
* m4
