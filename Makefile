PREFIX 	?= /usr/local
BINDIR  ?= $(PREFIX)/bin
SERVICE_USER    ?= localtimed
SERVICE_GROUP   ?= localtimed

TARGETS = localtimed localtime.service polkit.rules sysusers.conf

export GOPATH = $(CURDIR)

all: $(TARGETS)

clean:
	-rm -f $(TARGETS)

install-user:
	useradd -r -U localtimed

install: all
	install -Dm755 localtimed $(DESTDIR)$(BINDIR)/localtimed
	install -Dm640 polkit.rules $(DESTDIR)$(PREFIX)/share/polkit-1/rules.d/40-localtime.rules
	install -Dm644 localtime.service $(DESTDIR)$(PREFIX)/lib/systemd/system/localtime.service
	install -Dm644 sysusers.conf $(DESTDIR)$(PREFIX)/lib/sysusers.d/localtime.conf

%: %.in
	m4 -DBINDIR="$(BINDIR)" \
		-DPREFIX="$(PREFIX)" \
		-DUSER="$(SERVICE_USER)" \
		-DGROUP="$(SERVICE_GROUP)" \
		$< > $@

%: %.go
	go build -o $@ $<
