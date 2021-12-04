# themr

A program to set themes in all your programs by replacing strings in their config files.

# Usage

Just run

```
themr THEME
```

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
  -d, --debug         print debug messages
```

# Installation

```sh
make install
```

and optionally, copy over the example configs and themes

```sh
make examples
```

# Configs

Configs are stored in ~/.config/themr/configs.yaml as dictionaries containing the following:

* path: the path to the config file to edit
* regex: a regular expression used to find the line to replace
* pre: text that will go before the replaced string
* post: text that will go after the replaced string
* cmd: a shell command to run after making the replacement (optional)

## Example

```yaml
vim:
    path: "~/.config/nvim/init.vim"
    regex: 'colorscheme .*'
    pre: "colorscheme "
    post: ""
```

# Themes

Themes are stored in ~/.config/themr/themes.yaml as dictionaries containing key value pairs with
the name of the theme for each kind of config. 

A default name can be defined so that you don't have to repeat the name for each config if it's the same.
If a default name is not defined you *must* define a name for all your defined configs.

## Example:

```yaml
gruvbox:
    vim: "gruvbox8_hard"
    default: "gruvbox"
    # these two are not needed since they match the default
    # lightline: "gruvbox"
    # xresources: "gruvbox"
```
