//config.go
package main

import (
	"errors"

	ini "github.com/go-ini/ini"
)

var (
	configModtime  int64
	errNotModified = errors.New("Not modified")
)

// Config - структура для считывания конфигурационного файла
type Config struct {
	Epsilon          float64 //точность обработки чисел с плавающей запятой, не используется
	LogLevel         string
	DicFile          string
	Path             string
	pathToRepaire    string
	Comand           string
	Null             float64
	NullReplace      bool
	verifyDate       bool
	lasInfoReport    string
	lasCheckReport   string
	lasMessageReport string
	logMissingReport string
	maxWarningCount  int
}

////////////////////////////////////////////////////////////
func readGlobalConfig(fileName string) (x *Config, err error) {
	x = new(Config)
	gini, err := ini.Load(globalConfigName)
	if err != nil {
		return nil, err
	}
	x.LogLevel = gini.Section("global").Key("loglevel").String()
	x.Epsilon, err = gini.Section("global").Key("epsilon").Float64()
	x.DicFile = gini.Section("global").Key("filedictionary").String()
	x.Path = gini.Section("global").Key("path").String()
	x.pathToRepaire = gini.Section("global").Key("pathToRepaire").String()
	x.Comand = gini.Section("global").Key("cmd").String()
	x.Null, err = gini.Section("global").Key("stdNull").Float64()
	x.NullReplace, err = gini.Section("global").Key("replaceNull").Bool()
	x.verifyDate, err = gini.Section("global").Key("verifyDate").Bool()
	x.lasInfoReport = gini.Section("global").Key("lasInfoReport").String()
	x.lasCheckReport = gini.Section("global").Key("lasCheckReport").String()
	x.lasMessageReport = gini.Section("global").Key("lasMessageReport").String()
	x.logMissingReport = gini.Section("global").Key("logMissingReport").String()
	x.maxWarningCount, err = gini.Section("global").Key("maxWarningCount").Int()
	return x, err
}
