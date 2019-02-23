// glas
// Copyright 2018 softlandia@gmail.com
// Обработка las файлов. Построение словаря и замена мнемоник на справочные

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/softlandia/xlib"
	"gopkg.in/ini.v1"
)

const (
	fileNameMnemonic = "mnemonic.ini"
	globalConfigName = "glas.ini"
	configFileName   = "las.ini"
)

var (
	//Cfg - global programm config
	Cfg *Config
	//Mnemonic - map of std mnemonic
	Mnemonic map[string]string
	//Dic - mnemonic substitution dictionary
	Dic map[string]string
)

//============================================================================
func main() {
	log.Println("start ", os.Args[0])
	//configuration & dictionaries are filled in here
	//initialize() stop programm if error occure
	initialize()
	fmt.Printf("init ok.\n")
	fmt.Printf("precision: %v\n", Cfg.Epsilon)
	fmt.Printf("debug level: %v\n", Cfg.LogLevel)
	fmt.Printf("dictionary file: %v\n", Cfg.DicFile)
	fmt.Printf("input path: %v\n", Cfg.Path)
	fmt.Printf("output path: %v\n", Cfg.pathToRepaire)
	fmt.Printf("command: %v\n", Cfg.Comand)
	fmt.Printf("std NULL parameter: %v\n", Cfg.Null)
	fmt.Printf("replace NULL: %v\n", Cfg.NullReplace)
	fmt.Printf("verify date: %v\n", Cfg.verifyDate)
	fmt.Printf("report files: '%s', '%s', '%s'\n", Cfg.logFailReport, Cfg.logMessageReport, Cfg.logGoodReport)
	fmt.Printf("missing log report: %s\n", Cfg.logMissingReport)
	fmt.Printf("warning report: %s\n", Cfg.lasWarningReport)

	fileList := make([]string, 0, 10)
	//makeFilesList() stop programm if error occure
	n := makeFilesList(&fileList, Cfg.Path)

	switch Cfg.Comand {
	case "test":
		TEST(n)
	case "convert":
		log.Println("convert code page: ")
		convertCodePage(&fileList)
	case "verify":
		log.Println("verify las:")
	case "repair":
		log.Println("repaire las:")
		repairLas(&fileList, &Dic, Cfg.Path, Cfg.pathToRepaire, Cfg.logMessageReport)
	case "info":
		log.Println("collect log info:")
		statisticLas(&fileList, &Dic, Cfg.logFailReport, Cfg.logMessageReport, Cfg.logGoodReport, Cfg.lasWarningReport, Cfg.logMissingReport)
	}
}

///////////////////////////////////////////
func verifyLas(fl *[]string) error {
	log.Printf("action 'verify' not define")
	return nil
}

///////////////////////////////////////////
func convertCodePage(fl *[]string) error {
	log.Printf("action 'convert' not define")
	return nil
}

////////////////////////////////////////////////////////////
//load std mnemonic
func readGlobalMnemonic(iniFileName string) (map[string]string, error) {
	iniMnemonic, err := ini.Load(iniFileName)
	if err != nil {
		log.Printf("error on load std mnemonic, check out file 'mnemonic.ini'\n")
		return nil, err
	}
	sec, err := iniMnemonic.GetSection("mnemonic")
	if err != nil {
		log.Printf("error on read 'mnemonic.ini'")
		return nil, err
	}
	x := make(map[string]string)
	for _, s := range sec.KeyStrings() {
		x[s] = sec.Key(s).Value()
	}
	if Cfg.LogLevel == "DEBUG" {
		log.Println("__mnemonics:")
		for k, v := range x {
			fmt.Printf("mnemonic: %s, desc: %s\n", k, v)
		}
	}
	return x, nil
}

////////////////////////////////////////////////////////////
//init programm, read config, ini files and init dictionary
//stop programm if not successful
func initialize() {
	var err error

	//read global config from yaml file
	Cfg, err = readGlobalConfig(configFileName)
	if err != nil {
		log.Printf("Fail read '%s' config file. %v", configFileName, err)
		os.Exit(1)
	}

	Mnemonic, err = readGlobalMnemonic(fileNameMnemonic)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//read dectionary from ini file
	iniDic, err := ini.Load(Cfg.DicFile)
	if err != nil {
		log.Printf("Fail to read file: %v\n", err)
		os.Exit(2)
	}
	sec, err := iniDic.GetSection("LOG")
	if err != nil {
		log.Println("Fail load section 'LOG' from file 'dic.ini'. ", err)
		os.Exit(3)
	}
	//fill dictionary
	Dic = make(map[string]string)
	for _, s := range sec.KeyStrings() {
		Dic[s] = sec.Key(s).Value()
	}
	//словарь заполнен
	if Cfg.LogLevel == "DEBUG" {
		log.Println("__dic:")
		for k, v := range Dic {
			fmt.Println("key: ", k, " val: ", v)
		}
	}
}

//----------------------------------------------------------------------------
//makeFilesList - find and load to array all founded las files
func makeFilesList(fileList *[]string, path string) int {
	n, err := xlib.FindFilesExt(fileList, Cfg.Path, ".las")
	if err != nil {
		log.Println("error at search files. verify path: ", Cfg.Path, err)
		log.Println("stop")
		os.Exit(4)
	}
	if n == 0 {
		log.Println("files 'las' not found. verify parameter path: '", Cfg.Path, "' and change in 'main.yaml'")
		log.Println("stop")
		os.Exit(5)
	}
	if Cfg.LogLevel == "INFO" {
		log.Println("founded ", n, " las files:")
		if Cfg.LogLevel == "DEBUG" {
			for i, s := range *fileList {
				log.Println(i, " : ", s)
			}
		}
	}
	return n
}

//TEST - test read and write las files
func TEST(m int) {
	fmt.Printf("founded :%v las files", m)

	//test file "1.las"
	las := NewLas()
	//las.setFromCodePage(Cfg.CodePage)
	n, err := las.Open("1.las")
	if n == 7 {
		fmt.Println("TEST read 1.las OK")
		fmt.Println(err)
	} else {
		fmt.Println("TEST read 1.las ERROR")
		fmt.Println(err)
	}

	err = las.setNull(Cfg.Null)
	fmt.Println("set new null value done, error: ", err)

	err = las.Save("-1.las")
	if err != nil {
		fmt.Println("TEST save -1.las ERROR: ", err)
	} else {
		fmt.Println("TEST save -1.las OK")
	}

	las = nil
	las = NewLas()
	n, err = las.Open("-1.las")
	if (n == 7) && (las.Null == -999.25) {
		fmt.Println("TEST read -1.las OK")
		fmt.Println(err)
	} else {
		fmt.Println("TEST read -1.las ERROR")
		fmt.Println("NULL not -999.25 or count dept points != 7")
		fmt.Println(err)
	}

	las = nil
	las = NewLas(xlib.Cp866)
	n, err = las.Open("2.las")
	if n == 4895 {
		fmt.Println("TEST read 2.las OK")
		fmt.Println(err)
	} else {
		fmt.Println("TEST read 2.las ERROR")
		fmt.Println(err)
	}
	err = las.Save("-2.las")
	if err != nil {
		fmt.Println("TEST save -2.las ERROR")
		fmt.Println(err)
	} else {
		fmt.Println("TEST save -2.las OK")
	}
	las = nil

	las = NewLas(xlib.CpWindows1251)
	_, err = las.Open("4.las")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	err = las.Save("-4.las")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	oFile, _ := os.Create(Cfg.lasWarningReport)
	defer oFile.Close()
	for i, w := range las.warnings {
		fmt.Fprintf(oFile, "%d, dir: %d,\tsec: %d,\tl: %d,\tdesc: %s\n", i, w.direct, w.section, w.line, w.desc)
	}
	las = nil
}
