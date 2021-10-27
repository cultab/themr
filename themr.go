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

func main() {

    parser := argparse.NewParser("themr", "Set a theme in multiple programs by replacing strings in their config files.", &argparse.ParserConfig{DisableDefaultShowHelp: true})

    // parser.add_argument('theme_name', metavar='THEME', type=str, nargs='?', help='theme to use')
    // parser.add_argument('-l', '--list-themes', action='store_true', help="list supported themes and exit")
    // parser.add_argument('-c', '--list-configs', action='store_true', help='list supported configs and exit')
    // parser.add_argument('-d', '--debug', action='store_true', help='print debug messages')
    chosen_theme_name := parser.String("", "theme", &argparse.Option{Positional: true})
    list_configs_flag := parser.Flag("c", "list-configs", &argparse.Option{Help: "list supported configs"})
    list_themes_flag := parser.Flag("l", "list-themes", &argparse.Option{Help: "list supported themes"})
    debug := parser.Flag("d", "debug", &argparse.Option{Help: "pring debug messages"})

    if e := parser.Parse(os.Args[1:]); e != nil {
        fmt.Println("error:" + e.Error())
        return
    }

    config_dir, err := os.UserConfigDir()
    config_dir += "/themr/"
    if err != nil {
        fmt.Println("Could not determing User Config Directory. (is $HOME unset?)")
        os.Exit(1)
    }
    configs, err := load_configs(config_dir)
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }

    var config_names []string
    for _, config := range configs {
        config_names = append(config_names, config["name"])
    }

    themes, err := load_themes(config_dir, config_names)
    if err != nil {
        fmt.Println(err.Error())
        os.Exit(1)
    }

    if *list_configs_flag {
        list_configs(configs)
    }

    if *list_themes_flag {
        list_themes(themes)
    }

    if *debug { //do nothing
    }

    if *chosen_theme_name == "" {
        fmt.Println("No theme name given")
        os.Exit(1)
    }

    var chosen_theme conf = nil
    for _, theme := range themes {
        if theme["name"] == *chosen_theme_name {
            chosen_theme = theme
        }
    }

    if chosen_theme == nil {
         fmt.Println("No such theme exists")
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
            wg.Done() // defer is overkill here
        }(theme, config)
    }
    wg.Wait()
}

func set_config_theme(theme conf, config conf) {

    path := config["path"]
    if strings.HasPrefix(path, "~") {
        usr, _ := user.Current()
        path = filepath.Join(usr.HomeDir, path[2:])
        // fmt.Println(path)
    }

    regex, err := regexp.Compile(config["regex"])
    if err != nil {
        fmt.Println("Could not parse regex in config for " + config["name"])
    }
    // fmt.Println(regex)

    name := theme[config["name"]]
    if name == "" {
        name = theme["default"]
    }

    cmd, cmd_exists := config["cmd"]

    file, err := os.ReadFile(path)
    if err != nil {
        switch err {
        case os.ErrNotExist:
            fmt.Println("File not found error: '" + path + "' was not found")
        case os.ErrPermission:
            fmt.Println("Permission error: Not allowed to read '" + path + "'")
        default:
            fmt.Println("Error: Cannot open '" + path + "'")
            fmt.Println(err)
        }
        return
        // return nil, errors.New("Could not read '" + path + "'")
    }
    if !regex.Match(file) {
        fmt.Println("Welllll fuck")
        fmt.Println(config["name"])
        println(name)
        println(config["regex"])
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
        // just start it and let it fuck off
        // don't wait for it to finish
        // TODO: maybe add a switch to make it wait and print it's output for debuging?
        command.Start()
    }
}

func list_themes(themes []conf) {
    fmt.Println("Found themes:")

    for _, theme := range themes {
        fmt.Println("\t" + theme["name"])
    }
}

func list_configs(configs []conf) {
    fmt.Println("Found configs:")

    for _, theme := range configs {
        fmt.Println("\t" + theme["name"])
    }
}

func load_configs(config_dir string) ([]conf, error) {
    config_path := config_dir + "configs.yaml"
    configs := make(map[string]conf)

    file, err := os.ReadFile(config_path)
    if err != nil {
        switch err {
        case os.ErrNotExist:
            fmt.Println("File not found error: '" + config_path + "' was not found")
        case os.ErrPermission:
            fmt.Println("Permission error: Not allowed to read '" + config_path + "'")
        default:
            fmt.Println("Error: Cannot open '" + config_path + "'")
            fmt.Println(err)
        }
        return nil, errors.New("Could not read '" + config_path + "'")
    }

    yaml.Unmarshal(file, &configs)

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
        switch err {
        case os.ErrNotExist:
            fmt.Println("File not found error: '" + theme_path + "' was not found")
        case os.ErrPermission:
            fmt.Println("Permission error: Not allowed to read '" + theme_path + "'")
        default:
            fmt.Println("Error: Cannot open '" + theme_path + "'")
            fmt.Println(err)
        }
        return nil, errors.New("Could not read '" + theme_path + "'")
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
