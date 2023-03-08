package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"github.com/hellflame/argparse"
	"gopkg.in/yaml.v3"
	"themr/config"
)

var (
	logger log.Logger
	debug  *bool
)

func init() {
	logger = log.New(log.WithTimeFormat(""))
}

func main() {
	parser := argparse.NewParser("themr", "Set a theme in multiple programs by replacing strings in their config files.", &argparse.ParserConfig{DisableDefaultShowHelp: true})

	chosen_theme_name := parser.String("", "theme", &argparse.Option{Positional: true})
	list_configs_flag := parser.Flag("c", "list-configs", &argparse.Option{Help: "list supported configs"})
	list_themes_flag := parser.Flag("l", "list-themes", &argparse.Option{Help: "list supported themes"})
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

	// get config path
	config_dir, err := os.UserConfigDir()
	config_dir += "/themr/"
	if err != nil {
		logger.Error("Could not determine User Config Directory. (is $HOME unset?)")
		os.Exit(1)
	}

	configs, err := config.Load_configs(config_dir)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	// valid config types is the set resulting from
	// the union of config names and their types
	config_types := make(set)
	for _, config := range configs {
		config_types[config.Name] = member
		if config_type := config.Type; config_type != "" {
			config_types[config_type] = member
		}
	}

	// load themes
	themes, err := load_themes(config_dir, config_types)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	if *list_configs_flag {
		list_configs(configs)
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

	chosen_theme.set(configs)
}

func (theme theme_info) set(all_configs []config.Config) {

	// only keep configs with appropriate types
	var configs []config.Config
	for _, conf := range all_configs {
		if theme.Map().contains_key(conf.Type) {
			configs = append(configs, conf)
		}
	}

	if *debug {
		for _, config := range configs {
			theme.set_for(config)
		}
		return
	}

	var wg sync.WaitGroup
	for _, conf := range configs {
		wg.Add(1)
		go func(theme theme_info, conf config.Config) {
			defer wg.Done()
			theme.set_for(conf)
		}(theme, conf)
	}
	wg.Wait()
}

func (theme theme_info) set_for(config config.Config) {

	path := config.Path
	if strings.HasPrefix(path, "~") {
		usr, _ := user.Current()
		path = filepath.Join(usr.HomeDir, path[2:])
	}

	// use theme name for the type of config
	theme_name := theme[config.Type]
	// unless it's overwitten by a theme specifying a theme_name for a config
	if name, exists := theme[config.Name]; exists {
		theme_name = name
	}

	file, err := os.ReadFile(path)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	if !config.Regex.Match(file) {
		logger.Error("Configuration: Regex `" + config.Regex.String() + "` failed to match a line for " + theme_name)
		return
	}

	line := strings.ReplaceAll(config.Replace, "{}", theme_name)
	new_contents := config.Regex.ReplaceAll(file, []byte(line))

	// try to use the same Permission bits, just in case
	file_stat, err := os.Stat(path)
	if err != nil {
		logger.Error(err.Error())
	}
	// write back the file :)
	err = os.WriteFile(path, new_contents, file_stat.Mode())
	if err != nil {
		logger.Error(err.Error())
	}

	config.RunCmd(theme_name, *debug)
}

func list_themes(themes []theme_info) {
	fmt.Println("Found themes:")

	for _, theme := range themes {
		fmt.Println("\t" + theme["name"])
	}
}

func list_configs(configs []config.Config) {
	fmt.Println("Found configs:")

	for _, config := range configs {
		fmt.Println("\t" + config.Name)
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
