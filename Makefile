PREFIX = /usr/local

install:
	cp ./themr ${PREFIX}/bin/themr
	chmod +x ${PREFIX}/bin/themr

examples:
	cp -n ./example_configs.yaml ~/.config/themr/configs.yaml
	cp -n ./example_themes.yaml ~/.config/themr/themes.yaml

uninstall:
	rm -f ${PREFIX}/bin/themr

.PHONY: install uninstall examples
