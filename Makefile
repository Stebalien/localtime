PREFIX 	?= /usr/local
BINDIR  ?= $(PREFIX)/bin

TARGETS = localtimed

export GOPATH = $(CURDIR)

all: $(TARGETS)

clean:
	-rm -f $(TARGETS)

install: all
	install -Dm755 localtimed $(DESTDIR)$(BINDIR)/localtimed
	install -Dm640 data/polkit.rules $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/40-localtime.rules
	install -Dm644 data/localtime.service $(DESTDIR)$(PREFIX)/lib/systemd/system/localtime.service
	install -Dm644 data/sysusers.conf $(DESTDIR)$(PREFIX)/lib/sysusers.d/localtime.conf

%: %.go
	go build -o $@ $<
