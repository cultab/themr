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

	"github.com/hellflame/argparse"
	"gopkg.in/yaml.v3"
)

type conf map[string]string

var (
    debug *bool
)

func main() {

    // init parser
    parser := argparse.NewParser("themr", "Set a theme in multiple programs by replacing strings in their config files.", &argparse.ParserConfig{DisableDefaultShowHelp: true})

    // get args
    chosen_theme_name := parser.String("", "theme", &argparse.Option{Positional: true})
    list_configs_flag := parser.Flag("c", "list-configs", &argparse.Option{Help: "list supported configs"})
    list_themes_flag := parser.Flag("l", "list-themes", &argparse.Option{Help: "list supported themes"})
    debug = parser.Flag("d", "debug", &argparse.Option{Help: "pring debug messages"})

    if e := parser.Parse(os.Args[1:]); e != nil {
        fmt.Fprintln(os.Stderr, "error:" + e.Error())
        return
    }

    // get config path
    config_dir, err := os.UserConfigDir()
    config_dir += "/themr/"
    if err != nil {
        fmt.Fprintln(os.Stderr, "Could not determing User Config Directory. (is $HOME unset?)")
        os.Exit(1)
    }
    // load configs
    configs, err := load_configs(config_dir)
    if err != nil {
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(1)
    }

    // add config name into each config
    var config_names []string
    for _, config := range configs {
        config_names = append(config_names, config["name"])
    }

    // load themes
    themes, err := load_themes(config_dir, config_names)
    if err != nil {
        fmt.Fprintln(os.Stderr, err.Error())
        os.Exit(1)
    }

    if *list_configs_flag {
        list_configs(configs)
    }

    if *list_themes_flag {
        list_themes(themes)
    }

    if *list_configs_flag || *list_themes_flag {
        os.Exit(0)
    }

    if *chosen_theme_name == "" {
        fmt.Fprintln(os.Stderr, "No theme name given")
        os.Exit(1)
    }

    var chosen_theme conf
    for _, theme := range themes {
        if theme["name"] == *chosen_theme_name {
            chosen_theme = theme
        }
    }

    if chosen_theme == nil {
         fmt.Fprintln(os.Stderr, "No such theme exists")
         os.Exit(1)
    }

    set_theme(chosen_theme, configs)

}

func set_theme(theme conf, configs []conf) {
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
        // fmt.Fprintln(os.Stderr, path)
    }

    regex, err := regexp.Compile(config["regex"])
    if err != nil {
        fmt.Fprintln(os.Stderr, "Could not parse regex in config for " + config["name"])
    }
    // fmt.Fprintln(os.Stderr, regex)

    name := theme[config["name"]]
    if name == "" {
        name = theme["default"]
    }

    cmd, cmd_exists := config["cmd"]

    file, err := os.ReadFile(path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            err = fmt.Errorf("File not found: '" + path + "' was not found")
        } else if errors.Is(err, os.ErrPermission) {
            err = fmt.Errorf("Permission error: Not allowed to read '" + path + "'")
        } else {
            err = fmt.Errorf("Could not read '" + path + "'", err)
        }
        fmt.Fprintln(os.Stderr, err)
        return
    }
    if !regex.Match(file) {
        fmt.Fprintln(os.Stderr, "Configuration error: Regex failed to match a line in config for " + name)
        fmt.Fprintln(os.Stderr, "regex referenced: " + config["regex"])
    }
    new_contents := regex.ReplaceAll(file, []byte(config["pre"] + name + config["post"]))

    // try to use the same Permission bits, just in case
    file_stat, _ := os.Stat(path)
    // write back the file :)
    os.WriteFile(path, new_contents, file_stat.Mode())

    if cmd_exists {
        if strings.ContainsAny(cmd, "%") {
            cmd = strings.Replace(cmd, "%", name, 1)
        }
        command := exec.Command("sh", "-c", cmd)
        // just start it and let it fuck off, don't wait for it to finish
        // TODO: maybe add a switch to make it wait and print it's output for debuging?
        command.Start()
    }
}

func list_themes(themes []conf) {
    fmt.Fprintln(os.Stdout, "Found themes:")

    for _, theme := range themes {
        fmt.Fprintln(os.Stdout, "\t" + theme["name"])
    }
}

func list_configs(configs []conf) {
    fmt.Fprintln(os.Stdout, "Found configs:")

    for _, theme := range configs {
        fmt.Fprintln(os.Stdout, "\t" + theme["name"])
    }
}

func load_configs(config_dir string) ([]conf, error) {
    config_path := config_dir + "configs.yaml"
    configs := make(map[string]conf)

    file, err := os.ReadFile(config_path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return nil, fmt.Errorf("File not found error: '" + config_path + "' was not found")
        } else if errors.Is(err, os.ErrPermission) {
            return nil, fmt.Errorf("Permission error: Not allowed to read '" + config_path + "'")
        }
        return nil, fmt.Errorf("Could not read '" + config_path + "'", err)
    }

    err = yaml.Unmarshal(file, &configs)
    if err != nil {
        return nil, fmt.Errorf("Unmarshaling error: '" + config_path + "' was invalid")
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

func load_themes(config_dir string, required_configs []string) ([]conf, error) {
    theme_path := config_dir + "themes.yaml"
    themes := make(map[string]conf)

    file, err := os.ReadFile(theme_path)

    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return nil, fmt.Errorf("File not found error: '" + theme_path + "' was not found")
        } else if errors.Is(err, os.ErrPermission) {
            return nil, fmt.Errorf("Permission error: Not allowed to read '" + theme_path + "'")
        }
        return nil, fmt.Errorf("Could not read '" + theme_path + "'", err)
    }

    yaml.Unmarshal(file, &themes)

    var themes_list []conf

    for theme_name, theme := range themes {
        theme["name"] = theme_name

        if theme["default"] == "" {
            res, missing := theme.contains_all_keys(required_configs)
            if !res {
                return nil, errors.New("Missing key(s): [" + strings.Join(missing, ", ") + "] in theme " + theme_name)
            }
        }

        themes_list = append(themes_list, theme)
    }

    return themes_list, err
}

func contains_string(container []string, s string) bool {
    for _, k := range container {
        if k == s {
            return true
        }
    }
    return false
}

func (config conf) contains_key(key string) bool {
    for k := range config {
        if k == key {
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
