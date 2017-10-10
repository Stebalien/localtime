PREFIX 	?= /usr/local
BINDIR  ?= $(PREFIX)/bin
SERVICE_USER    ?= localtimed

TARGETS = localtimed localtime.service polkit.rules

export GOPATH = $(CURDIR)

.PHONY: all clean install-user install

all: $(TARGETS)

clean:
	-rm -f $(TARGETS)

install-user:
	useradd -r -U localtimed

install: all
	install -Dm755 localtimed $(DESTDIR)$(BINDIR)/localtimed
	install -Dm640 polkit.rules $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/40-localtime.rules
	install -Dm644 localtime.service $(DESTDIR)$(PREFIX)/lib/systemd/system/localtime.service

%: %.in
	m4 -DBINDIR="$(BINDIR)" \
		-DPREFIX="$(PREFIX)" \
		-DUSER="$(SERVICE_USER)" \
		$< > $@

%: %.go
	go build -o $@ $<
