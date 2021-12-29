PREFIX = /usr/local

build:
	go build themr.go

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

clean:
	rm --force themr

.PHONY: install uninstall examples
