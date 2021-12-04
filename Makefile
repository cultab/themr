PREFIX = /usr/local

build:
	go build themr.go

install:
	cp ./themr ${PREFIX}/bin/themr
	chmod +x ${PREFIX}/bin/themr

examples:
	cp -n ./example_configs.yaml ~/.config/themr/configs.yaml
	cp -n ./example_themes.yaml ~/.config/themr/themes.yaml

uninstall:
	rm -f ${PREFIX}/bin/themr

clean:
	rm -f themr

.PHONY: install uninstall examples
