# Makefile for a go project
#
# Author: Jon Eisen
# 	site: joneisen.me
# 	
# Targets:
# 	all: Builds the code
# 	build: Builds the code
# 	fmt: Formats the source files
# 	clean: cleans the code
# 	install: Installs the code to the GOPATH
# 	iref: Installs referenced projects
#	test: Runs the tests
#	
#  Blog post on it: http://joneisen.me/post/25503842796
#

# Go parameters

DESTDIR ?= ""
PREFIX ?= "/usr"

TARGETS = tzupdated

all: $(TARGETS)

clean:
	-rm -f $(TARGETS)

install: all
	install -Dm755 tzupdated $(DESTDIR)$(PREFIX)/bin/tzupdated
	install -Dm644 data/polkit.rules $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/40-tzupdate.rules
	install -Dm644 data/tzupdate.service $(DESTDIR)$(PREFIX)/lib/systemd/system/tzupdate.service
	install -Dm644 data/sysusers.conf $(DESTDIR)$(PREFIX)/lib/sysusers.d/tzupdate.conf

%: %.go
	go build -o $@ $<
