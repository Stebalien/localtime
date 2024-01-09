PREFIX 	?= /usr/local
BINDIR  ?= $(PREFIX)/bin
SERVICE_USER    ?= localtimed

GO111MODULE = on
TARGETS = localtimed localtime.service localtime.sysusers polkit.rules geoclue-demo-agent.service

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
	install -Dm644 localtime.sysusers $(DESTDIR)$(PREFIX)/lib/sysusers.d/localtime.conf
	install -Dm644 geoclue-demo-agent.service $(DESTDIR)$(PREFIX)/lib/systemd/system/geoclue-demo-agent.service

%: %.in
	m4 -DBINDIR="$(BINDIR)" \
		-DPREFIX="$(PREFIX)" \
		-DUSER="$(SERVICE_USER)" \
		$< > $@

%: %.go
	go build -o $@ $<
