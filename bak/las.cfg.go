//config for class Las
//storte settings
package main

import (
	"log"
	"os"

	ini "gopkg.in/ini.v1"
)

//TLasCfg - class to store global Las settings
type TLasCfg struct {
	null          float64
	outputCharSet int
}

//LasCfg - global var to store Las config
var LasCfg TLasCfg

//InitLasCfg - read ini file and init las settings
func InitLasCfg() {
	//read las ini configuration ini file
	logCfg, err := ini.Load(lasCfgFileName)
	if err != nil {
		log.Printf("Fail to read file: %v\n", err)
		os.Exit(1)
	}
	if Cfg.LogLevel == "DEBUG" {
		log.Printf("Las ini file open success\n")
	}

	LasCfg.null, err = logCfg.Section("global").Key("null").Float64()
	if err != nil {
		log.Println("Fail load section 'LOG' from file 'dic.ini'. ", err)
		os.Exit(2)
	}
	if Cfg.LogLevel == "DEBUG" {
		log.Printf("std NULL value: %v\n", LasCfg.null)
	}

	LasCfg.outputCharSet, err = logCfg.Section("global").Key("outputcharset").Int()
	if err != nil {
		log.Println("Fail load section 'LOG' from file 'dic.ini'. ", err)
		os.Exit(2)
	}
	if Cfg.LogLevel == "DEBUG" {
		log.Printf("output char set: %v\n", LasCfg.outputCharSet)
	}
}
