PREFIX 	?= /usr/local
BINDIR  ?= $(PREFIX)/bin

TARGETS = tzupdated

export GOPATH = $(CURDIR)

all: $(TARGETS)

clean:
	-rm -f $(TARGETS)

install: all
	install -Dm755 tzupdated $(DESTDIR)$(BINDIR)/tzupdated
	install -dm750 $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/
	install -m640 data/polkit.rules $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/40-tzupdate.rules
	install -Dm644 data/tzupdate.service $(DESTDIR)$(PREFIX)/lib/systemd/system/tzupdate.service
	install -Dm644 data/sysusers.conf $(DESTDIR)$(PREFIX)/lib/sysusers.d/tzupdate.conf
	
	chgrp polkitd $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/40-tzupdate.rules

%: %.go
	go build -o $@ $<
