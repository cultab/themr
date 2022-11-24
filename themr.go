package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/hellflame/argparse"
	"gopkg.in/yaml.v3"
)

type conf map[string]string

type set map[string]struct{}

var (
    member struct{}
    debug *bool
    Log *log.Logger
    Error *log.Logger
    Debug *log.Logger
)

func main() {

    Log = log.New(os.Stdout, "", 0)
    Error = log.New(os.Stderr, "", 0)

    // init parser
    parser := argparse.NewParser("themr", "Set a theme in multiple programs by replacing strings in their config files.", &argparse.ParserConfig{DisableDefaultShowHelp: true})

    // get args
    chosen_theme_name := parser.String("", "theme", &argparse.Option{Positional: true})
    list_configs_flag := parser.Flag("c", "list-configs", &argparse.Option{Help: "list supported configs"})
    list_themes_flag := parser.Flag("l", "list-themes", &argparse.Option{Help: "list supported themes"})
    debug = parser.Flag("d", "debug", &argparse.Option{Help: "pring debug messages"})

    if *debug {
        Debug = log.New(os.Stderr, "", 0)
    } else {
        Debug = log.New(io.Discard, "", 0)
    }

    if e := parser.Parse(os.Args[1:]); e != nil {
        Error.Fatal("error:" + e.Error())
    }

    // get config path
    config_dir, err := os.UserConfigDir()
    config_dir += "/themr/"
    if err != nil {
        Error.Fatal("Could not determing User Config Directory. (is $HOME unset?)")
    }
    // load configs
    configs, err := load_configs(config_dir)
    if err != nil {
        Error.Fatal(err.Error())
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
        Error.Fatal(err.Error())
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
        Error.Fatal("No theme name given")
    }

    var chosen_theme conf
    for _, theme := range themes {
        if theme["name"] == *chosen_theme_name {
            chosen_theme = theme
        }
    }

    if chosen_theme == nil {
         Error.Fatal("No such theme exists")
    }

    set_theme(chosen_theme, configs)

}

func set_theme(theme conf, all_configs []conf) {

    // only keep configs with appropriate types
    var configs[] conf
    for _, config := range all_configs {
        if theme.contains_key(config["type"]) {
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

    regex, err := regexp.Compile(config["regex"])
    if err != nil {
        Error.Println(fmt.Errorf("Could not parse regex for " + config["name"] + ": %w", err))
        return
    }

    // use theme name for the type of config
    theme_name := theme[config["type"]]
    // unless it's overwitten by a theme specifying a theme_name for a config
    if name, exists := theme[config["name"]]; exists {
        theme_name = name
    }

    file, err := os.ReadFile(path)
    if err != nil {
        Error.Println(err)
        return
    }
    if !regex.Match(file) {
        Error.Println("Configuration error: Regex failed to match a line for " + theme_name)
        Error.Println("regex referenced: " + config["regex"])
        return
    }
    new_contents := regex.ReplaceAll(file, []byte(config["pre"] + theme_name + config["post"]))

    // try to use the same Permission bits, just in case
    file_stat, _ := os.Stat(path)
    // write back the file :)
    os.WriteFile(path, new_contents, file_stat.Mode())

    if cmd, exists := config["cmd"]; exists {
        if strings.ContainsAny(cmd, "%") {
            cmd = strings.Replace(cmd, "%", theme_name, 1)
        }
        command := exec.Command("sh", "-c", cmd)

        if *debug {
            Debug.Println("Running command for " + config["name"] + ": " + cmd)
            out, err := command.CombinedOutput()
            if err != nil {
                Debug.Println(fmt.Errorf("Command for " + config["name"] + " failed: %w", err))
            }
            Debug.Println(string(out))
        } else {
            // just start it and let it fuck off, don't wait for it to finish
            command.Start()
        }
    }
}

func list_themes(themes []conf) {
    Log.Println("Found themes:")

    for _, theme := range themes {
        Log.Println("\t" + theme["name"])
    }
}

func list_configs(configs []conf) {
    Log.Println("Found configs:")

    for _, theme := range configs {
        Log.Println("\t" + theme["name"])
    }
}

func load_configs(config_dir string) ([]conf, error) {
    config_path := config_dir + "configs.yaml"
    configs := make(map[string]conf)

    file, err := os.ReadFile(config_path)
    if err != nil {
        switch {
        case errors.Is(err, os.ErrNotExist):
            return nil, fmt.Errorf("File not found error: '" + config_path + "' was not found")
        case errors.Is(err, os.ErrPermission):
            return nil, fmt.Errorf("Permission error: Not allowed to read '" + config_path + "'")
        default:
            return nil, fmt.Errorf("Could not read '" + config_path + "'", err)
        }
    }

    err = yaml.Unmarshal(file, &configs)
    if err != nil {
        return nil, fmt.Errorf("Unmarshaling error: '" + config_path + "' was invalid because %w", err)
    }

    var configs_list []conf

    for config_name, config := range configs {
        config["name"] = config_name

        required_keys := []string{"path", "regex", "pre", "post"}

        res, missing := config.contains_all_keys(required_keys)
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
        if ! theme.contains_at_least_one_key(config_types) {
            return nil, fmt.Errorf("Theme must have at least one config type: '" + theme["name"] + "' does not!")
        }

        themes_list = append(themes_list, theme)
    }

    return themes_list, err
}

// LOL NO GENERICS {
func (config conf) contains_key(key interface{}) bool {
    for k := range config {
        if k == key {
            return true
        }
    }
    return false
}

func (key_set set) contains_key(key interface{}) bool {
    for k := range key_set {
        if k == key {
            return true
        }
    }
    return false
}
// } --> LOL NO GENERICS

func (config conf) contains_at_least_one_key(keys set) bool {
    for key := range config {
        if config.contains_key(key) {
            return true
        }
    }
    return false
}

func (config conf) contains_all_keys(keys []string) (bool, []string) {
    var not_contained []string

    for _, key := range keys {
        if !config.contains_key(key) {
            not_contained = append(not_contained, key)
        }
    }
    if len(not_contained) > 0 {
        return false, not_contained
    }

    return true, nil
}
