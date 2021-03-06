#!/bin/env python3
"""A program to set a global theme by replacing strings in config files."""

import re
import argparse
import yaml
import subprocess
from os import environ
from os.path import expanduser


debug = False


def main():
    """Entry point."""
    global debug

    parser = argparse.ArgumentParser(
        description="Set a theme in multiple programs by replacing strings in their config files."
    )

    parser.add_argument('theme_name', metavar='THEME', type=str, nargs='?', help='theme to use')
    parser.add_argument('-l', '--list-themes', action='store_true', help="list supported themes and exit")
    parser.add_argument('-c', '--list-configs', action='store_true', help='list supported configs and exit')
    parser.add_argument('-d', '--debug', action='store_true', help='print debug messages')

    args = parser.parse_args()

    debug = args.debug
    chosen_theme_name = args.theme_name

    config_folder_path = environ.get("XDG_CONFIG_HOME", expanduser("~/.config"))
    config_folder_path += "/themr"

    configs = load_configs(config_folder_path)
    themes, invalid_themes = load_themes(config_folder_path, [name for name in configs.keys()])

    if args.list_themes:
        list_themes(themes)

    if args.list_configs:
        list_configs(configs)

    if args.list_themes or args.list_configs:
        exit(0)

    if chosen_theme_name is None:
        print("No theme name given")
        exit(1)
    elif chosen_theme_name in invalid_themes:
        print(f"Theme {chosen_theme_name} exists but was ignored due to the errors above.")
        exit(1)

    if (chosen_theme := themes.get(chosen_theme_name)) is None:
        print(f'Theme "{chosen_theme_name}" does not exist')
        exit(1)

    set_theme(chosen_theme, configs)


def load_themes(folder_path, config_names):
    """Load and validate themes from file."""

    invalid_themes = list()

    try:
        with open(f'{folder_path}/themes.yaml') as themes_stream:
            themes = yaml.safe_load(themes_stream)  # TODO: use a class for themes
    except FileNotFoundError:
        print("themes.yaml was not found!")
        exit(-1)

    for theme_name, theme in dict(themes).items():
        valid = True

        if not theme.get("default"):
            for config_name in config_names:
                if config_name not in theme.keys():
                    print(f'Missing value for "{config_name}" in {theme_name} theme, while a default value is not given.')

                    valid = False

            if not valid:
                print(f'Ignored theme: {theme_name}.')
                themes.pop(theme_name)
                invalid_themes.append(theme_name)

    return themes, invalid_themes


def load_configs(folder_path):
    """Load and validate configs from file."""
    try:
        with open(expanduser(f'{folder_path}/configs.yaml')) as configs_stream:
            configs = yaml.safe_load(configs_stream)  # TODO: use a class for themes
    except FileNotFoundError:
        print("configs.yaml was not found!")
        exit(-1)

    required_keys = ['path', 'regex', 'pre', 'post']

    for config_name, config in dict(configs).items():
        # make name available like the rest of the values
        config["name"] = config_name

        valid = True

        for key in required_keys:
            if key not in config.keys():
                print(f'Missing value for "{key}" in {config["name"]} config.')

                valid = False

        if not valid:
            print(f'Ignoring {config_name} config.')
            configs.pop(config_name)

    return configs


def list_themes(themes):
    global debug

    print("Found themes:")
    for theme in themes:
        print("\t" + theme)

    if debug:
        print(themes)


def list_configs(configs):
    global debug

    print("Found configs:")
    for config in configs:
        print("\t" + config)

    if debug:
        print(configs)


def get_new_config_contents(theme, config):
    warnings = list()
    success = False

    config_name = config["name"]
    config_path = config["path"]
    regex = re.compile(config["regex"])

    # If we don't have a specific theme name, use the default.
    # This is done so if a theme has the same name in every config,
    # you don't have to repeat it each time.
    replace = theme.get(config_name, theme["default"])

    new_config_contents = list()
    try:
        with open(expanduser(config_path), 'r') as config_file:
            for line in config_file:
                if regex.match(line) and not success:
                    new_config_contents.append(config["pre"] + replace + config["post"] + '\n')
                    if debug:
                        print(f'found line:  {line}', end='')
                        print(f'replaced by: {config["pre"] + replace + config["post"]}')
                    success = True
                else:
                    new_config_contents.append(line)
    except FileNotFoundError:
        warnings.append(f'No file found at: "{config["path"]}')

    if not success:
        warnings.append(f'No line matched regex: "{config["regex"]}"')

    return new_config_contents, warnings


def write_new_config_file(config, contents):
    config_path = expanduser(config["path"])

    with open(config_path, 'w') as config_file:
        for line in contents:
            config_file.write(line)


def set_theme_for_config(theme, config):
    warnings = list()

    if debug:
        print(f'Working on {config["name"]}')

    new_config_contents, warn = get_new_config_contents(theme, config)

    if not warn:
        write_new_config_file(config, new_config_contents)
    else:
        warnings.extend(warn)

    try:
        cmd = config["cmd"]
        replace = theme.get(config["name"], theme["default"])

        if '%' in cmd:
            cmd = cmd.replace("%", replace)

        subprocess.Popen(cmd, shell=True, stdin=None, stdout=None)

        if debug:
            print(f'Executing post replace command: "{cmd}"')

    except KeyError:
        pass

    if warnings:
        print(f'Warnings from {config["name"]} config:')
        for warn in warnings:
            print('\t' + warn + '\n')

    return warnings


def set_theme(theme, configs):
    """Set the theme in configs."""
    global debug

    for config_name, config in configs.items():

        warnings = set_theme_for_config(theme, config)

        if warnings:
            print(f'Warnings from {config_name} config:')
            for warn in warnings:
                print('\t' + warn + '\n')


if __name__ == "__main__":
    main()
