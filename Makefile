PREFIX = /usr/local

install:
	cp ./themr ${PREFIX}/themr

examples:
	cp -n ./example_configs.yaml ~/.config/themr/configs.yaml
	cp -n ./example_themes.yaml ~/.config/themr/themes.yaml

uninstall:
	rm -f ${PREFIX}/themr

.PHONY: install uninstall examples
