# themr

A program to set themes in all your programs by replacing strings in their config files.

# Usage

As easy as `themr THEME`.

Where THEME is a theme you defined in ~/.config/themr/themes.yaml

```
usage: themr [-h] [-l] [-c] [-d] [THEME]

Set a theme in multiple programs by replacing strings in their config files.

positional arguments:
  THEME               theme to use

optional arguments:
  -h, --help          show this help message and exit
  -l, --list-themes   list supported themes and exit
  -c, --list-configs  list supported configs and exit
  -v, --version       print version
  -d, --debug         print debug messages
```

# Installation

```sh
make install
```
Includes zsh completion script, installed to `$PREFIX/share/zsh/site-functions/`

Optionally, copy over the example configs and themes

```sh
make examples
```

# Configs

Configs are stored in $XDG_CONFIG_HOME/themr/configs.yaml as dictionaries containing the following:

* path: the path to the config file to edit
* regex: a regular expression used to find the line to replace
* replace: text that will go inplace of the old line
* type: the type of config, eg: colorscheme, polybar config. (optional)
* cmd: a shell command to run after making the replacement (optional)
* create: whether to create a file if it's missing (optional, defaults to false)

## Example configs.yaml

```yaml
# name of the config
vim:
    path: "~/.config/nvim/init.vim" # path to the file to edit
    regex: 'colorscheme .*' # regex to find the line to edit (should only match once)
    replace: "colorscheme {}" # what to use instead of instead of the old line found with the regex above
    type: colorscheme # (optional defaults to the name of the config), type the config
    cmd: "echo {}" # (optional) a command to run
    create: false # (optional, defaults to false) whether to create a file if it's missing
```

The placeholder `{}` is replaced by the theme name at runtime.

# Themes

Themes are stored in `$XDG_CONFIG_HOME/themr/themes.yaml` as dictionaries containing key value pairs with
the name of the theme for each kind of config.

A default name can be defined so that you don't have to repeat the name for each config if it's the same.
If a default name is not defined you *must* define a name for all your defined configs.

# Examples

This defines a theme of type `colorscheme` that is named `gruvbox`, there is one special case for configs of type `nvim`, where the name of the theme becomes `gruvbox8_hard`. You would apply this theme with `themr gruvbox`.

```yaml
gruvbox:
    nvim: "gruvbox8_hard"   # special case for nvim
    colorscheme: "gruvbox" # default for configs of type colorscheme

    # the following is not needed since it matches the default for configs of type `colorscheme`
    # xresources: "gruvbox"
```

Some of the configs used by this theme are as follows:


* The `xresources` config is used because it's type `colorscheme`, matches one of the keys of the theme `gruvbox`.

* The regex looks for lines that begin with `#include "` and are followed by a filename ending in `.xdefaults`.

* The replacement line is defined as `#include "` then the theme name with `.xdefaults` appended.

* The create flag is set because it just so happens this file contains only this one line and is itself `#include`'ed by another file.

* A command is defined `reload_xrdb`, which is a script that runs `xrdb -merge` appropriately.

```yaml
xresources:
  type: colorscheme
  path: "~/.config/xrdb/selected_theme.xdefaults"
  regex: '#include ".*\.xdefaults"'
  replace: '#include "./{}.xdefaults"'
  cmd: "reload_xrdb"
  create: true
```

Again the type matches, but the theme has a special case defined for this config.
As such the replacement line will contain `return "<the special case of the theme>"`.

```yaml
nvim:
  type: colorscheme
  path: "~/.config/nvim/lua/user/colorscheme.lua"
  regex: 'return ".*"'
  replace: 'return "{}"'
  create: true
  cmd: "all_nvim 'colorscheme {}'"
```

This is a theme that uses many config definitions to achive a rounder ui look.

```yaml
round ui:
  polybar: "round_transparent_modules"
  picom: 15
  bspwm-single_monocle: false
  bspwm-window_gap: 16
  dunst-corner: 15
  dunst-offset: "30x80"
```

The config definitions used are as follows:

```yaml
picom:
  path: "~/.config/picom.conf"
  regex: ".*corner-radius = .*# themr #"
  replace: "corner-radius = {} # themr #"

bspwm-single_monocle:
  path: "~/.config/bspwm/bspwmrc"
  regex: ".* single_monocle .* # themr:bspwm-single_monocle #"
  replace: "bspc config single_monocle {} # themr:bspwm-single_monocle #"
  cmd: "bspc config single_monocle {}"

bspwm-window_gap:
  path: "~/.config/bspwm/bspwmrc"
  regex: ".* window_gap .* # themr:bspwm-window_gap #"
  replace: "bspc config window_gap {} # themr:bspwm-window_gap #"
  cmd: "bspc config window_gap {}"

dunst-corner:
  path: "~/.config/dunst/no_theme.dunstrc"
  regex: "corner_radius = .*"
  replace: "corner_radius = {}"

dunst-offset:
  path: "~/.config/dunst/no_theme.dunstrc"
  regex: "offset = .*"
  replace: "offset = {}"
  cmd: "reload_dunst"
```
