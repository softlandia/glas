// Copyright 2018 softlandia@gmail.com
// Обработка las файлов. Построение словаря и замена мнемоник на справочные

package main

import (
	//	"errors"
	//	"io"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

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
	fmt.Printf("path: %v\n", Cfg.Path)
	fmt.Printf("command: %v\n", Cfg.Comand)
	fmt.Printf("std NULL parameter: %v\n", Cfg.Null)
	fmt.Printf("replace NULL: %v\n", Cfg.NullReplace)
	fmt.Printf("verify date: %v\n", Cfg.verifyDate)
	fmt.Printf("report files: '%s', '%s', '%s'\n", Cfg.logFailReport, Cfg.logPooreReport, Cfg.logGoodReport)
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
		repairLas(&fileList, &Dic, Cfg.Path, Cfg.pathToRepaire)
	case "info":
		log.Println("collect log info:")
		statisticLas(&fileList, &Dic, Cfg.logFailReport, Cfg.logPooreReport, Cfg.logGoodReport, Cfg.lasWarningReport, Cfg.logMissingReport)
	}
}

func repaireOneFile(signal chan int, las *Las, inputFolder, folderOutput string, flag *int, msg *string) {
	if las == nil {
		*flag = 0
		*msg = "las is nil"
		signal <- 1
		return
	}
	n, err := las.Open(las.FileName)
	if las.Wraped() {
		*flag = 1
		*msg = fmt.Sprintf("las file %s ignored, WRAP=YES\n", las.FileName)
		signal <- 1
		return
	}
	if (n == 0) && (err != nil) {
		*flag = 1
		*msg = fmt.Sprintf("on las file %s, occure error: %v file ignore\n", las.FileName, err)
		signal <- 1
		return
	}
	las.FileName = strings.Replace(las.FileName, inputFolder, folderOutput, 1)

	err = las.Save(las.FileName, true)
	if err != nil {
		*flag = 1
		*msg = "error on save file: " + las.FileName
		signal <- 1
		return
	}
	*flag = 0
	*msg = ""
	signal <- 1
	return
}

func repaireOneFileListener(signal chan int, count int, tStart time.Time) {
	n := 0
	for {
		n += (<-signal)
		if n >= count {
			break
		}
		switch n {
		case 100:
		case 250:
		case 500:
		case 1000:
			log.Printf("%d files done, elapsed: %v\n", n, time.Since(tStart))
		}
	}
	log.Printf("%d files done, all done elapsed: %v\n press Enter", n, time.Since(tStart))
}

///////////////////////////////////////////
//1. read las
//2. save las to new folder
func repairLas(fl *[]string, dic *map[string]string, inputFolder, folderOutput string) error {
	if len(*fl) == 0 {
		return errors.New("las files for repaire not ready")
	}
	var signal chan int = make(chan int)
	go repaireOneFileListener(signal, len(*fl), time.Now())
	for _, f := range *fl {
		las := NewLas()
		las.LogDic = &Mnemonic
		las.VocDic = &Dic
		las.FileName = f
		flag := 0
		msg := ""
		go repaireOneFile(signal, las, inputFolder, folderOutput, &flag, &msg)
		if flag == 1 {
			fmt.Println(msg)
		}
		las = nil
	}
	var s string
	fmt.Scanln(&s)
	return nil
}

func statLas(signal chan int, oFile, wFile *os.File, missingMnemonic map[string]string, f string) {
	las := NewLas()
	las.LogDic = &Mnemonic
	las.VocDic = &Dic
	n, err := las.Open(f)

	//amt := time.Duration(rand.Intn(250))
	//time.Sleep(time.Millisecond * amt)

	//write warnings
	if len(las.warnings) > 0 {
		wFile.WriteString("#file: " + las.FileName + "\n")
		for i, w := range las.warnings {
			fmt.Fprintf(wFile, "%d, dir: %d,\tsec: %d,\tl: %d,\tdesc: %s\n", i, w.direct, w.section, w.line, w.desc)
		}
		wFile.WriteString("\n")
	}
	if las.Wraped() {
		fmt.Printf("las file %s ignored, WRAP=YES\n", f)
		las = nil
		signal <- 1
		return
	}

	if (n == 0) && (err != nil) {
		fmt.Printf("on las file %s, occure error: %v file ignore\n", f, err)
		las = nil
		signal <- 1
		return
	}

	fmt.Fprintf(oFile, "#logs in file: '%s':\n", f)
	for k, v := range las.Logs {
		if len(v.Mnemonic) == 0 {
			fmt.Fprintf(oFile, "*input log: %s \t internal: %s \t mnemonic:%s*\n", v.iName, k, v.Mnemonic)
			missingMnemonic[v.iName] = v.iName
		} else {
			fmt.Fprintf(oFile, "input log: %s \t internal: %s \t mnemonic: %s\n", v.iName, k, v.Mnemonic)
		}
	}
	fmt.Fprintf(oFile, "\n")
	las = nil
	signal <- 1
}

func statLasListener(signal chan int, count int, tStart time.Time) {
	n := 0
	for {
		n += (<-signal)
		if n == count {
			break
		}
	}
	log.Printf(" prosess done, elapsed: %v\n press Enter", time.Since(tStart))
}

///////////////////////////////////////////
//1. формируется список каротажей не имеющих словарной мнемоники - logMissingReport
//2. формируется список ошибочных файлов - write to console (using log.)
//3. формируется отчёт о предупреждениях при прочтении las файлов - lasWarningReport
//4. формируется отчёт прочитанных файлах, для каких каротажей найдена подстановка, для каких нет - reportFail
func statisticLas(fl *[]string, dic *map[string]string, reportFail, reportPoor, reportGood, lasWarningReport, logMissingReport string) error {
	var missingMnemonic map[string]string
	missingMnemonic = make(map[string]string)
	var signal chan int = make(chan int)

	log.Printf("make log statistic")
	if len(*fl) == 0 {
		return errors.New("file to statistic not found")
	}
	oFile, err := os.Create(reportFail)
	if err != nil {
		log.Print("report file: '", reportFail, "' not open to write, ", err)
		return err
	}
	defer oFile.Close()
	oFile.WriteString("###list of logs\n")

	wFile, _ := os.Create(lasWarningReport)
	defer wFile.Close()
	wFile.WriteString("#list of warnings\n")

	go statLasListener(signal, len(*fl), time.Now())
	for _, f := range *fl {
		go statLas(signal, oFile, wFile, missingMnemonic, f)
	}
	var s string
	fmt.Scanln(&s)

	mFile, _ := os.Create(logMissingReport)
	defer mFile.Close()
	mFile.WriteString("missing log\n")
	keys := make([]string, 0, len(missingMnemonic))
	for k := range missingMnemonic {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		mFile.WriteString(missingMnemonic[k] + "\n")
	}
	return nil
}

///////////////////////////////////////////
func verifyLas(fl *[]string) error {
	log.Printf("action not define")
	return nil
}

///////////////////////////////////////////
func convertCodePage(fl *[]string) error {
	log.Printf("action not define")
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
