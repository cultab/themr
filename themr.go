package main

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"runtime"

	"themr/config"

	"github.com/charmbracelet/log"
	"github.com/hellflame/argparse"
	"gopkg.in/yaml.v3"
)

var (
	logger log.Logger
	debug  *bool
)

const VERSION = "0.2.4"

func init() {
	logger = log.New(log.WithTimeFormat(""))
}

func main() {
	parser := argparse.NewParser("themr", "Set a theme in multiple programs by replacing strings in their config files.", &argparse.ParserConfig{DisableDefaultShowHelp: true})

	chosen_theme_name := parser.String("", "theme", &argparse.Option{Positional: true})
	list_configs_flag := parser.Flag("c", "list-configs", &argparse.Option{Help: "list supported configs"})
	list_themes_flag := parser.Flag("l", "list-themes", &argparse.Option{Help: "list supported themes"})
	print_version := parser.Flag("v", "version", &argparse.Option{Help: "print version"})
	debug = parser.Flag("d", "debug", &argparse.Option{Help: "print debug messages"})

	if e := parser.Parse(nil); e != nil {
		// check if empty error, happens with -h flag
		if msg := e.Error(); msg != "" {
			logger.Error(e.Error())
		}
		os.Exit(1)
	}

	if *debug {
		logger.SetLevel(log.DebugLevel)
	}

	config.SetLogger(logger)

	if *print_version {
		fmt.Println(os.Args[0], "v"+VERSION)
		os.Exit(0)
	}

	// get config path
	config_dir, err := os.UserConfigDir()
	config_dir += "/themr/"
	if err != nil {
		logger.Error("Could not determine User Config Directory. (is $HOME unset?)")
		os.Exit(1)
	}

	// Override config directory to use ~/.config on macOS
	if runtime.GOOS == "darwin" {
			config_dir = filepath.Join(os.Getenv("HOME"), ".config")
			config_dir += "/themr/"
	}

	if *debug {
		logger.Debug(config_dir)
	}
	edits, err := config.Load_configs(config_dir)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	// valid config types is the set resulting from
	// the union of config names and their types
	config_types := make(set)
	var str_configs string
	for _, configs := range edits {
		for _, config := range configs {
			str_configs += config.Name + "\n"
			config_types[config.Name] = member
			if config_type := config.Type; config_type != "" {
				config_types[config_type] = member
			}
		}
	}
	logger.Debug(fmt.Sprintf("loaded %d configs", len(edits)), "names", str_configs)

	// load themes
	themes, err := load_themes(config_dir, config_types)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	var str_themes string
	for _, theme := range themes {
		str_themes += theme["name"] + "\n"
	}
	logger.Debug(fmt.Sprintf("loaded %d themes", len(themes)), "names", str_themes)

	if *list_configs_flag {
		list_configs(edits)
		os.Exit(0)
	}

	if *list_themes_flag {
		list_themes(themes)
		os.Exit(0)
	}

	if *chosen_theme_name == "" {
		logger.Error("No theme name given")
		os.Exit(1)
	}

	var chosen_theme theme_info
	for _, theme := range themes {
		if theme["name"] == *chosen_theme_name {
			chosen_theme = theme
		}
	}

	if chosen_theme == nil {
		logger.Error("No such theme exists")
		os.Exit(1)
	}

	chosen_theme.set(edits)
}

func (t theme_info) set(edits config.Edits) {

	// only keep configs with appropriate types
	// var configs []config.Config
	for path, configs := range edits {
		for i, config := range configs {
			if !t.Map().contains_key(config.Type) {
				if len(edits[path]) == 1 { // if only one config left for this edit
					delete(edits, path)
				} else { // else remove only the 1 config
					edits[path] = append(edits[path][:i], edits[path][i+1:]...)
				}
			}
		}
	}

	if *debug {
		for _, configs := range edits {
			for _, config := range configs {
				t.set_for(config)
			}
		}
		return
	}

	var wg sync.WaitGroup
	for _, configs := range edits {
		wg.Add(1)
		go func(t theme_info, configs []config.Config) {
			for _, conf := range configs {
				defer wg.Done()
				t.set_for(conf)
			}
		}(t, configs)
	}
	wg.Wait()
}

func (t theme_info) set_for(config config.Config) {

	path := config.Path
	if strings.HasPrefix(path, "~") {
		usr, _ := user.Current()
		path = filepath.Join(usr.HomeDir, path[2:])
	}

	// use theme name for the type of config
	theme_name := t[config.Type]
	// unless it's overwitten by a theme specifying a theme_name for a config
	if name, exists := t[config.Name]; exists {
		theme_name = name
	}

	// warn if directory of config does not exist
	_, err := os.Stat(filepath.Dir(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Path to config does not exist.\n\tMaybe you forgot to stow something?")
		} else {
			logger.Error("Stat() config dir: " + err.Error())
		}
	}

	file, err := os.ReadFile(path)
	// if the config tells use to create the file if it doesn't exist
	// we just place the "Replace" string in the "file"'s contents
	// and write those back
	if err != nil {
		if !config.Create {
			logger.Error("Can't read file: " + err.Error())
			return
		}
		file = []byte(config.Replace)
	}
	if !config.Regex.Match(file) {
		logger.Error("Configuration: Regex `" + config.Regex.String() + "` failed to match in file: " + config.Path)
		return
	}

	line := strings.ReplaceAll(config.Replace, "{}", theme_name)
	new_contents := config.Regex.ReplaceAll(file, []byte(line))

	// create path to config, incase iy does not exist
	// err = os.MkdirAll(filepath.Dir(path), os.FileMode(0700))
	// if err != nil {
		// logger.Error("Can't create directory path to config: " + err.Error())
	// }

	// write back the file :)
	err = os.WriteFile(path, new_contents, os.FileMode(0664))
	if err != nil {
		logger.Error("Can't save: " + err.Error())
	}

	err = config.RunCmd(theme_name, *debug)
	if err != nil {
		logger.Warn(fmt.Errorf("Command for "+config.Name+" failed: %w", err).Error())
	}
}

func list_themes(themes []theme_info) {
	fmt.Println("Found themes:")

	for _, theme := range themes {
		fmt.Println("\t" + theme["name"])
	}
}

func list_configs(edits config.Edits) {
	fmt.Println("Found configs:")

	for _, configs := range edits {
		for _, config := range configs {
			fmt.Println("\t" + config.Name)
		}
	}
}

func load_themes(config_dir string, config_types set) ([]theme_info, error) {
	theme_path := config_dir + "themes.yaml"
	themes := make(map[string]theme_info)

	file, err := os.ReadFile(theme_path)

	if err != nil {
		return nil, err
	}

	yaml.Unmarshal(file, &themes)

	var themes_list []theme_info

	for theme_name, theme := range themes {
		theme["name"] = theme_name

		// check if theme contains at least one config type
		if !theme.Map().contains_at_least_one_key(config_types) {
			return nil, fmt.Errorf("Theme must have at least one config type: '" + theme["name"] + "' does not!")
		}

		themes_list = append(themes_list, theme)
	}

	return themes_list, err
}
