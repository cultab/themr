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
* type: the type of config, eg colorscheme, polybar config. (optional)
* cmd: a shell command to run after making the replacement (optional)

## Example configs.yaml

```yaml
vim:
    path: "~/.config/nvim/init.vim" # path to the file to edit
    regex: 'colorscheme .*'         # regex to find the line to edit
    replace: "colorscheme {}"       # what to put instead of the old line,
    type: colorscheme               # type of config, (if empty the name is used, in this example "vim")
    cmd: "echo {}"                  # a command to run (optional)
```

The placeholder `{}` is replaced by the theme name at runtime.

# Themes

Themes are stored in $XDG_CONFIG_HOME/themr/themes.yaml as dictionaries containing key value pairs with
the name of the theme for each kind of config.

A default name can be defined so that you don't have to repeat the name for each config if it's the same.
If a default name is not defined you *must* define a name for all your defined configs.

## Example themes.yaml

```yaml
gruvbox:
    vim: "gruvbox8_hard"   # special case for vim
    colorscheme: "gruvbox" # default for configs of type colorscheme

    # these two are not needed since they match the default for configs of type `colorscheme`
    # lightline: "gruvbox"
    # xresources: "gruvbox"
```
