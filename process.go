// glas
// Copyright 2018 softlandia@gmail.com
// Обработка las файлов. Построение словаря и замена мнемоник на справочные
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar"
	"github.com/softlandia/glasio"
)

//1. read las
//2. save las to new folder
func repaireOneFile(signal chan int, las *glasio.Las, inputFolder, folderOutput string, wFile *os.File, messages *[]string, wg *sync.WaitGroup) {
	defer wg.Done()
	if las == nil {
		*messages = append(*messages, "las is nil")
		signal <- 1
		return
	}
	n, err := las.Open(las.FileName)
	las.SaveWarningToFile(wFile)

	if las.IsWraped() {
		*messages = append(*messages, fmt.Sprintf("las file %s ignored, WRAP=YES\n", las.FileName))
		signal <- 1
		return
	}
	if n == 0 {
		*messages = append(*messages, fmt.Sprintf("on las file %s, **error**: %v file ignore\n", las.FileName, err))
		signal <- 1
		return
	}
	if err != nil {
		//non critical error, continue
		*messages = append(*messages, fmt.Sprintf("on las file %s, *warning*: %v\n", las.FileName, err))
	}

	las.FileName = strings.Replace(las.FileName, inputFolder, folderOutput, 1)

	err = las.Save(las.FileName, true)
	if err != nil {
		*messages = append(*messages, "error on save file: "+las.FileName+" :: ")
		*messages = append(*messages, err.Error()+"\n")
		signal <- 1
		return
	}
	signal <- 1
}

func repaireOneFileListener(signal chan int, count int, wg *sync.WaitGroup) {
	n := 0
	bar := progressbar.New(count)
	bar.RenderBlank() // will show the progress bar
	for {
		n += (<-signal)
		bar.Add(1)
		if n >= count {
			wg.Done()     //заканчивается ЭТА горутина
			fmt.Println() //прогресс-бар отрисован, новая строка
			break
		}
	}
}

///////////////////////////////////////////
func repairLas(fl *[]string, dic *map[string]string, inputFolder, folderOutput, messageReport, warningReport string) error {
	if len(*fl) == 0 {
		return errors.New("las files for repaire not ready")
	}
	log.Printf("files count: %d", len(*fl))

	var signal = make(chan int)
	var wg sync.WaitGroup

	warnFile, _ := os.Create(warningReport)
	defer warnFile.Close()

	messages := make([]string, 0, len(*fl))
	tStart := time.Now()

	wg.Add(1)
	go repaireOneFileListener(signal, len(*fl), &wg)

	for _, f := range *fl {
		wg.Add(1)
		las := glasio.NewLas()
		las.LogDic = &Mnemonic
		las.VocDic = &Dic
		las.FileName = f
		go repaireOneFile(signal, las, inputFolder, folderOutput, warnFile, &messages, &wg)
		las = nil
	}
	wg.Wait()
	log.Printf("all done, elapsed: %v\n", time.Since(tStart))
	lFile, err := os.Create(messageReport)
	defer lFile.Close()
	if err == nil {
		for _, msg := range messages {
			lFile.WriteString(msg)
		}
	}
	return nil
}

func (m *tMessages) msgFileIsWraped(fn string) string {
	return fmt.Sprintf("file '%s' ignore, WRAP=YES\n", fn)
}

func (m *tMessages) msgFileNoData(fn string) string {
	return fmt.Sprintf("*error* file '%s', no data read ,*ignore*\n", fn)
}

func (m *tMessages) msgFileOpenWarning(fn string, err error) string {
	return fmt.Sprintf("**warning** file '%s' : %v **passed**\n", fn, err)
}

const (
	statLasCheck_OPEN_WRN = 30   // - open return warning
	statLasCheck_WRNG     = 1000 // - after this value warning is IMPORTANT, дальнейшая работа с файлом нежелательна
	statLasCheck_WRAP     = 1010 // - WRAP is ON
	statLasCheck_DATA     = 1020 // - data not readed
)

// LasLog - store logging info about las, fills up info from las.open()
type LasLog struct {
	filename        string              // file to read
	readedNumPoints int                 // number points readed from file
	errorOnOpen     error               // result from las.open()
	msgOpen         glasio.TLasWarnings // сообщения формируемые в процессе открытия las файла
	msgCheck        tMessages           // информация об особых случаях, генерируется statLas()
	msgReport       tInfoReport         // информация о каждом методе хранящемся в LAS файле, записывается в "log.info.md"
	missMnemonic    tMMnemonic
}

// NewLasLog - lasLog constructor
func NewLasLog(filename string) LasLog {
	var lasLog LasLog
	lasLog.filename = filename
	lasLog.msgOpen = nil
	lasLog.msgCheck = make(tMessages, 0, 10)
	lasLog.msgReport = make(tInfoReport, 0, 10)
	lasLog.missMnemonic = make(tMMnemonic, 0)
	return lasLog
}

// считывает файл и собирает все сообщения в один объект
func lasOpenCheck(filename string) LasLog {
	lasLog := NewLasLog(filename)

	las := glasio.NewLas() // TODO make special constructor to initialize with global Mnemonic and Dic
	las.LogDic = &Mnemonic // global var
	las.VocDic = &Dic      // global var

	lasLog.readedNumPoints, lasLog.errorOnOpen = las.Open(filename)
	lasLog.msgOpen = las.Warnings

	if las.IsWraped() {
		lasLog.msgCheck = append(lasLog.msgCheck, lasLog.msgCheck.msgFileIsWraped(filename))
		//return statLasCheck_WRAP
	}
	if las.NumPoints() == 0 {
		lasLog.msgCheck = append(lasLog.msgCheck, lasLog.msgCheck.msgFileNoData(filename))
		//return statLasCheck_DATA
	}
	if lasLog.errorOnOpen != nil {
		lasLog.msgCheck = append(lasLog.msgCheck, lasLog.msgCheck.msgFileOpenWarning(filename, lasLog.errorOnOpen))
	}

	for k, v := range las.Logs {
		if len(v.Mnemonic) == 0 { //v.Mnemonic содержит автоопределённую стандартную мнемонику, если она пустая, значит пропущена, помечаем **
			lasLog.msgReport = append(lasLog.msgReport, fmt.Sprintf("*input log: %s \t internal: %s \t mnemonic:%s*\n", v.IName, k, v.Mnemonic))
			lasLog.missMnemonic[v.IName] = v.IName
		} else {
			lasLog.msgReport = append(lasLog.msgReport, fmt.Sprintf("input log: %s \t internal: %s \t mnemonic: %s\n", v.IName, k, v.Mnemonic))
		}
	}

	las = nil
	return lasLog
}

// messages - slice of messages, generates from las.Open()
// wFile - file to write warnings
func statLas(signal chan int, wg *sync.WaitGroup, fileName string, lasLogger *LasLogger) {
	defer wg.Done()
	lasLogger.add(lasOpenCheck(fileName))
	signal <- 1
}

func statLasListener(signal chan int, count int, wg *sync.WaitGroup) {
	n := 0
	p := progressbar.New(count)
	p.RenderBlank()
	for {
		n += (<-signal)
		p.Add(1)
		if n == count {
			fmt.Println()
			wg.Done()
			break
		}
	}
}

///////////////////////////////////////////
//1. формируется список каротажей не имеющих словарной мнемоники - logMissingReport
//2. формируется список ошибочных файлов - write to console (using log.)
//3. формируется отчёт о предупреждениях при прочтении las файлов - lasWarningReport
//4. формируется отчёт прочитанных файлах, для каких каротажей найдена подстановка, для каких нет - fileInfoReport
func statisticLas(fl *[]string, dic *map[string]string, cfg *Config) error {
	log.Printf("make log statistic")
	if len(*fl) == 0 {
		return errors.New("files to statistic not found")
	}
	var signal = make(chan int)
	lasLogger := make(LasLogger, 0, len(*fl))
	var wg sync.WaitGroup
	tStart := time.Now()
	wg.Add(1)
	go statLasListener(signal, len(*fl), &wg)
	for _, f := range *fl {
		wg.Add(1)
		go statLas(signal, &wg, f, &lasLogger)
	}
	wg.Wait()
	lasLogger.save(cfg)
	log.Printf("info done, elapsed: %v\n", time.Since(tStart))
	return nil
}

// LasLogger - store messages from all las files
type LasLogger []LasLog

func (l *LasLogger) add(lasLog LasLog) {
	*l = append(*l, lasLog)
}

func (l LasLogger) save(cfg *Config) error {
	msgFile, err := os.Create(cfg.lasMessageReport)
	if err != nil {
		return fmt.Errorf("report file: '%s' not open to write: %v", cfg.lasMessageReport, err)
	}
	defer msgFile.Close()
	msgFile.WriteString("#MESSAGES#\n")
	msgFile.WriteString("##данные сообщения генерируются при чтении las файла, это различные проблемы связанные со структурой файла##\n\n")

	checkFile, err := os.Create(cfg.lasCheckReport)
	if err != nil {
		return fmt.Errorf("report file: '%s' not open to write: %v", cfg.lasCheckReport, err)
	}
	defer checkFile.Close()
	checkFile.WriteString("#WARNINGS#\n")
	checkFile.WriteString("##сообщения о существенных проблемах с файлом##\n")

	infoFile, err := os.Create(cfg.lasInfoReport)
	if err != nil {
		return fmt.Errorf("report file: '%s' not open to write: %v", cfg.lasInfoReport, err)
	}
	defer infoFile.Close()
	infoFile.WriteString("#list of logs#\n\n")

	missFile, err := os.Create(cfg.logMissingReport)
	if err != nil {
		return fmt.Errorf("report file: '%s' not open to write: %v", cfg.logMissingReport, err)
	}
	defer missFile.Close()
	missFile.WriteString("#missing logs#\n\n")

	for _, v := range l {
		msgFile.WriteString("**file: " + v.filename + "**\n")
		v.msgOpen.SaveWarningToFile(msgFile)
		v.msgCheck.save(checkFile)
		v.missMnemonic.save(missFile)
		v.msgReport.save(infoFile, v.filename)
	}
	return nil
}

// store messages from las.Open()
type tMessages []string

func (m *tMessages) save(f *os.File) {
	for _, msg := range *m {
		f.WriteString(msg)
	}
}

type tInfoReport []string

func (ir *tInfoReport) save(f *os.File, filename string) {

	fmt.Fprintf(f, "##logs in file: '%s'##\n", filename)
	for _, s := range *ir {
		f.WriteString(s)
	}
	f.WriteString("\n")
}

type tMMnemonic map[string]string

func (mm *tMMnemonic) save(f *os.File) {
	keys := make([]string, 0, len(*mm))
	for k := range *mm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f.WriteString((*mm)[k] + "\n")
	}
}
