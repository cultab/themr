package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/hellflame/argparse"
	"gopkg.in/yaml.v3"
)

type Map[K comparable, V any] map[K]V

type conf Map[string, string]

type set Map[string, struct{}]


var (
	member struct{}
	debug  *bool
)

// white
func Log(format string, a...interface{}) {
    c := color.New(color.FgWhite)
    fmt.Fprintln(os.Stdout, c.Sprintf(format, a...))
}

// red
func Error(format string, a...interface{}) {
    c := color.New(color.FgRed)
    fmt.Fprintln(os.Stderr, c.Sprintf(format, a...))
}

// cyan
func Debug(format string, a...interface{}) {
    c := color.New(color.FgCyan)
    fmt.Fprintln(os.Stderr, c.Sprintf(format, a...))
}


func main() {
	parser := argparse.NewParser("themr", "Set a theme in multiple programs by replacing strings in their config files.", &argparse.ParserConfig{DisableDefaultShowHelp: true})

	chosen_theme_name := parser.String("", "theme", &argparse.Option{Positional: true})
	list_configs_flag := parser.Flag("c", "list-configs", &argparse.Option{Help: "list supported configs"})
	list_themes_flag := parser.Flag("l", "list-themes", &argparse.Option{Help: "list supported themes"})
	debug = parser.Flag("d", "debug", &argparse.Option{Help: "pring debug messages"})

	if e := parser.Parse(nil); e != nil {
        // check if empty error, happens with -h flag
        if msg := e.Error(); msg != "" {
            Error(e.Error())
        }
		os.Exit(1)
	}

	// get config path
	config_dir, err := os.UserConfigDir()
	config_dir += "/themr/"
	if err != nil {
		Error("Could not determine User Config Directory. (is $HOME unset?)")
		os.Exit(1)
	}

	configs, err := load_configs(config_dir)
	if err != nil {
		Error(err.Error())
		os.Exit(1)
	}

	// valid config types is the set resulting from
	// the union of config names and their types
	config_types := make(set)
	for _, config := range configs {
		config_types[config["name"]] = member
		if config_type, exists := config["type"]; exists {
			config_types[config_type] = member
		}
	}

	// load themes
	themes, err := load_themes(config_dir, config_types)
	if err != nil {
		Error(err.Error())
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
		Error("No theme name given")
		os.Exit(1)
	}

	var chosen_theme conf
	for _, theme := range themes {
		if theme["name"] == *chosen_theme_name {
			chosen_theme = theme
		}
	}

	if chosen_theme == nil {
		Error("No such theme exists")
		os.Exit(1)
	}

	set_theme(chosen_theme, configs)

}

func set_theme(theme conf, all_configs []conf) {

    // only keep configs with appropriate types
    var configs[] conf
    for _, config := range all_configs {
        if theme.Map().contains_key(config["type"]) {
            configs = append(configs, config)
        }
    }

	if *debug {
		for _, config := range configs {
			set_config_theme(theme, config)
		}
		return
	}

	var wg sync.WaitGroup
	for _, config := range configs {
		wg.Add(1)
		go func(theme conf, config conf) {
			set_config_theme(theme, config)
			defer wg.Done()
		}(theme, config)
	}
	wg.Wait()
}

func set_config_theme(theme conf, config conf) {

	path := config["path"]
	if strings.HasPrefix(path, "~") {
		usr, _ := user.Current()
		path = filepath.Join(usr.HomeDir, path[2:])
	}

	// use theme name for the type of config
	theme_name := theme[config["type"]]
	// unless it's overwitten by a theme specifying a theme_name for a config
	if name, exists := theme[config["name"]]; exists {
		theme_name = name
	}

	regex, err := regexp.Compile(config["regex"])
	if err != nil {
		Error(fmt.Errorf("Could not parse regex for "+config["name"]+": %w", err).Error())
		return
	}

	file, err := os.ReadFile(path)
	if err != nil {
		Error(err.Error())
		return
	}
	if !regex.Match(file) {
		Error("Configuration: Regex `" + config["regex"] + "` failed to match a line for " + theme_name)
		return
	}
	new_contents := regex.ReplaceAll(file, []byte(config["pre"]+theme_name+config["post"]))

	// try to use the same Permission bits, just in case
	file_stat, err := os.Stat(path)
	if err != nil {
		Error(err.Error())
	}
	// write back the file :)
	err = os.WriteFile(path, new_contents, file_stat.Mode())
	if err != nil {
		Error(err.Error())
	}

	run_command(config, theme_name)
}

func run_command(config conf, theme_name string) {
	cmd, exists := config["cmd"]

	if !exists {
		return
	}

	if strings.ContainsAny(cmd, "%") {
		cmd = strings.Replace(cmd, "%", theme_name, 1)
	}

	command := exec.Command("sh", "-c", cmd)

	// just start it and let it fuck off, don't wait for it to finish
	if *debug {
		Debug("Running command for " + config["name"] + ": " + cmd)
		out, err := command.CombinedOutput()
		if err != nil {
			Error(fmt.Errorf("Command for "+config["name"]+" failed: %w", err).Error())
		}
		Debug(string(out))
		return
	}

	command.Start()

}

func list_themes(themes []conf) {
	Log("Found themes:")

	for _, theme := range themes {
		Log("\t" + theme["name"])
	}
}

func list_configs(configs []conf) {
	Log("Found configs:")

	for _, theme := range configs {
		Log("\t" + theme["name"])
	}
}

func load_configs(config_dir string) ([]conf, error) {
	config_path := config_dir + "configs.yaml"
	configs := make(map[string]conf)

	file, err := os.ReadFile(config_path)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(file, &configs)
	if err != nil {
		return nil, fmt.Errorf("Unmarshaling error: '"+config_path+"' was invalid because %w", err)
	}

	var configs_list []conf

	for config_name, config := range configs {
		config["name"] = config_name

		required_keys := []string{"path", "regex", "pre", "post"}

        res, missing := config.Map().contains_all_keys(required_keys)
        if !res {
            return nil, errors.New("Missing key(s): [" + strings.Join(missing, ", ") + "] in config for " + config_name)
        }

		configs_list = append(configs_list, config)
	}

	return configs_list, err
}

func load_themes(config_dir string, config_types set) ([]conf, error) {
	theme_path := config_dir + "themes.yaml"
	themes := make(map[string]conf)

	file, err := os.ReadFile(theme_path)

	if err != nil {
		return nil, err
	}

	yaml.Unmarshal(file, &themes)

	var themes_list []conf

    for theme_name, theme := range themes {
        theme["name"] = theme_name
        
        // check if theme contains at least one config type
        if ! theme.Map().contains_at_least_one_key(config_types) {
            return nil, fmt.Errorf("Theme must have at least one config type: '" + theme["name"] + "' does not!")
        }

		themes_list = append(themes_list, theme)
	}

	return themes_list, err
}

// generic methods, kinda

func (c conf) Map() Map[string, string] {
    return Map[string,string](c)
}

func (s set) Map() Map[string, struct{}] {
    return Map[string,struct{}](s)
}

func (m Map[K, V]) contains_key(key K) bool {
    for k := range m {
        if k == key {
            return true
        }
    }
    return false
}

func (m Map[K, V]) contains_at_least_one_key(keys set) bool {
    for key := range m {
        if m.contains_key(key) {
            return true
        }
    }
    return false
}

func (m Map[K, V]) contains_all_keys(keys []K) (bool, []K) {
    var not_contained []K

    for _, key := range keys {
        if !m.contains_key(key) {
            not_contained = append(not_contained, key)
        }
    }
    if len(not_contained) > 0 {
        return false, not_contained
    }

	return true, nil
}
