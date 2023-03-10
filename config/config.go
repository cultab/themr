package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"gopkg.in/yaml.v3"
)

var logger log.Logger

func SetLogger(l log.Logger) {
	logger = l
}

// represents one config in config.yaml
type yamlConfig struct {
	Type    string `yaml:"type"`
	Path    string `yaml:"path"`
	Regex   string `yaml:"regex"`
	Replace string `yaml:"replace"`
	Cmd     string `yaml:"cmd"`
}

// represents the entire config.yaml file
type yamlConfigFile map[string]yamlConfig

// represents an actual Config of a program
type Config struct {
	Name    string
	Type    string
	Path    string
	Regex   regexp.Regexp
	Replace string
	Cmd     *exec.Cmd
}

// type alias for a slice of Config
type configs []Config

// validate that all the fields expected exist
func (c yamlConfig) Validate(name string) error {
	var missing []string
	if c.Path == "" {
		missing = append(missing, "path")
	}
	if c.Regex == "" {
		missing = append(missing, "regex")
	}
	if c.Replace == "" {
		missing = append(missing, "replace")
	}
    if ! strings.Contains(c.Replace, "{}") {
        return errors.New("missing '{}' placeholder for replacement line:\n"+c.Replace+"\nin config for " + name)
    }

	if len(missing) != 0 {
		return errors.New("missing key(s): [" + strings.Join(missing, ", ") + "] in config for " + name)
	}
	return nil
}

func (c *configs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	cf := yamlConfigFile{}
	unmarshal(&cf)

	for name, conf := range cf {
		if err := conf.Validate(name); err != nil {
			return err
		}
		regex, err := regexp.Compile(conf.Regex)
		if err != nil {
			return fmt.Errorf("Could not parse regex for "+name+": %w", err)
		}
		cmd := exec.Command("sh", "-c", conf.Cmd)
		*c = append([]Config(*c), Config{
			Name:    name,
			Type:    conf.Type,
			Path:    conf.Path,
			Regex:   *regex,
			Replace: conf.Replace,
			Cmd:     cmd,
		})
	}

	return nil
}

// Loads all the configs found in the config.yaml file
// found the directory path given by config_dir
// NOTE: The only exported function
func Load_configs(config_dir string) ([]Config, error) {
    if logger == nil {
        logger = log.New(log.WithLevel(log.DebugLevel))
    }

	config_path := config_dir + "configs.yaml"

	file, err := os.ReadFile(config_path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file: %v", err)
	}

	// NOTE: confs is of type `configs` just so we make it implement the Unmarshaller interface
	var confs configs
	err = yaml.Unmarshal(file, &confs)
	if err != nil {
		return nil, fmt.Errorf("error while loading config: '"+config_path+"' was invalid because %w", err)
	}

	return confs, err
}

func (c Config) RunCmd(theme_name string, debug bool) {
	if c.Cmd != nil {
		return
	}

	if debug {
		logger.Debug("Running command for " + c.Name + ": " + strings.Join(c.Cmd.Args, " "))

		out, err := c.Cmd.CombinedOutput()
		if err != nil {
			logger.Warn(fmt.Errorf("Command for "+c.Name+" failed: %w", err).Error())
		}
		msgs := strings.Split(string(out), "\n")
		for i, msg := range msgs[0 : len(msgs)-1] {
			logger.Debug(fmt.Sprintf("%5s", fmt.Sprintf("[%d]:", i)), msg)
		}
		return
	}

    // PERF: just start it and let it fuck off, don't wait for it to finish
	c.Cmd.Start()

}
