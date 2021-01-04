// conf
package conf

import (
	"encoding/json"
	// "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"fyne.io/fyne"
)

const (
	ConfFileName = "litebook.conf"
)

func ConfDir() string {
	userconfigdir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(userconfigdir, "litebook")
}

type Config struct {
	LastWinSize    fyne.Size
	LastFile       string
	Theme          *Look
	BackgroundFile string
}

func NewDefConfig() (cfg *Config, err error) {
	cfg = &Config{
		LastFile:       "",
		BackgroundFile: "",
		LastWinSize:    fyne.NewSize(1000, 800),
	}
	cfg.Theme, err = defaultLook()
	return
}

func LoadConfigFile() (*Config, error) {
	cnf, err := NewDefConfig()
	if err != nil {
		return nil, err
	}
	buf, err := ioutil.ReadFile(filepath.Join(ConfDir(), ConfFileName))
	if err != nil {
		return cnf, err
	}
	err = json.Unmarshal(buf, cnf)
	return cnf, err
}

func SaveConfigFile(cnf *Config) error {
	buf, err := json.MarshalIndent(cnf, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(ConfDir(), ConfFileName), buf, 0644)
}
