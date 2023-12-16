PREFIX = /usr/local
VERSION = $(shell grep -Eo '[0-9]+\.[0-9]+\.[0-9]+' themr.go)
MACHINE = $(shell uname --machine)

build:
	go build .

install:
	cp ./themr ${PREFIX}/bin/themr
	mkdir --parents ${PREFIX}/share/zsh/site-functions
	cp ./_themr ${PREFIX}/share/zsh/site-functions/_themr
	chmod +x ${PREFIX}/bin/themr

examples:
	cp --no-clobber ./example_configs.yaml ~/.config/themr/configs.yaml
	cp --no-clobber ./example_themes.yaml ~/.config/themr/themes.yaml

uninstall:
	rm --force ${PREFIX}/bin/themr
	rm --force ${PREFIX}/share/zsh/site-functions/_themr

binary-release: build
	tar -czf themr-${VERSION}-${MACHINE}.tar.gz themr

source-release:
	tar -czf themr_${VERSION}_source.tar.gz \
		themr.go \
		genericMap.go \
		config/config.go \
		go.mod \
		go.sum \
		_themr \
		Makefile \
		example_themes.yaml \
		example_configs.yaml \
		LICENSE \
		README.md

clean:
	rm --force themr

.PHONY: install uninstall examples binary-release source-release build
