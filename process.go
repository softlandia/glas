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
)

//1. read las
//2. save las to new folder
func repaireOneFile(signal chan int, las *Las, inputFolder, folderOutput string, messages *[]string, wg *sync.WaitGroup) {
	defer wg.Done()
	if las == nil {
		*messages = append(*messages, "las is nil")
		signal <- 1
		return
	}
	n, err := las.Open(las.FileName)
	if las.Wraped() {
		*messages = append(*messages, fmt.Sprintf("las file %s ignored, WRAP=YES\n", las.FileName))
		signal <- 1
		return
	}
	if (n == 0) && (err != nil) {
		*messages = append(*messages, fmt.Sprintf("on las file %s, occure error: %v file ignore\n", las.FileName, err))
		signal <- 1
		return
	}
	las.FileName = strings.Replace(las.FileName, inputFolder, folderOutput, 1)

	err = las.Save(las.FileName, true)
	if err != nil {
		*messages = append(*messages, "error on save file: "+las.FileName)
		signal <- 1
		return
	}
	//	*msg = ""
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
			wg.Done()
			fmt.Println()
			break
		}
	}
}

///////////////////////////////////////////
func repairLas(fl *[]string, dic *map[string]string, inputFolder, folderOutput, logFile string) error {
	if len(*fl) == 0 {
		return errors.New("las files for repaire not ready")
	}
	log.Printf("files count: %d", len(*fl))

	var signal chan int = make(chan int)
	var wg sync.WaitGroup

	messages := make([]string, 0, len(*fl))
	tStart := time.Now()

	wg.Add(1)
	go repaireOneFileListener(signal, len(*fl), &wg)

	for _, f := range *fl {
		wg.Add(1)
		las := NewLas()
		las.LogDic = &Mnemonic
		las.VocDic = &Dic
		las.FileName = f
		//msg := ""
		go repaireOneFile(signal, las, inputFolder, folderOutput, &messages, &wg)
		las = nil
	}
	wg.Wait()
	log.Printf("all done, elapsed: %v\n", time.Since(tStart))
	lFile, err := os.Create(logFile)
	defer lFile.Close()
	if err == nil {
		for _, msg := range messages {
			lFile.WriteString(msg)
		}
	}
	return nil
}

func statLas(signal chan int, wg *sync.WaitGroup, oFile, wFile *os.File, missingMnemonic map[string]string, f string, messages *[]string) {
	defer wg.Done()
	las := NewLas()
	las.LogDic = &Mnemonic
	las.VocDic = &Dic
	n, err := las.Open(f)

	//write warnings
	if len(las.warnings) > 0 {
		wFile.WriteString("#file: " + las.FileName + "\n")
		for i, w := range las.warnings {
			fmt.Fprintf(wFile, "%d, dir: %d,\tsec: %d,\tl: %d,\tdesc: %s\n", i, w.direct, w.section, w.line, w.desc)
		}
		wFile.WriteString("\n")
	}
	if las.Wraped() {
		*messages = append(*messages, fmt.Sprintf("las file %s ignored, WRAP=YES\n", f))
		las = nil
		signal <- 1
		return
	}

	if (n == 0) && (err != nil) {
		*messages = append(*messages, fmt.Sprintf("on las file %s, occure error: %v file ignore\n", f, err))
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
//4. формируется отчёт прочитанных файлах, для каких каротажей найдена подстановка, для каких нет - reportFail
func statisticLas(fl *[]string, dic *map[string]string, reportFail, reportLog, reportGood, lasWarningReport, logMissingReport string) error {
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

	lFile, _ := os.Create(reportLog)
	defer lFile.Close()
	lFile.WriteString("#messages from statisticLas()\n")

	var wg sync.WaitGroup
	tStart := time.Now()
	messages := make([]string, 0, len(*fl))

	wg.Add(1)
	go statLasListener(signal, len(*fl), &wg)

	for _, f := range *fl {
		wg.Add(1)
		go statLas(signal, &wg, oFile, wFile, missingMnemonic, f, &messages)
	}
	wg.Wait()
	log.Printf("info done, elapsed: %v\n", time.Since(tStart))

	for _, msg := range messages {
		lFile.WriteString(msg)
	}

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