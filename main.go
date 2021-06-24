package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: ./scriptor script.yaml")
	}
	f, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	script := &Script{}
	if err := yaml.Unmarshal(f, script); err != nil {
		log.Fatal(err)
	}
	script.sender()
}

type Script struct {
	Blocks []Block `yaml:"blocks"`
}

type Block struct {
	Text          string         `yaml:"text"`
	Header        string         `yaml:"header"`
	Footer        string         `yaml:"fooder"`
	HeaderDelay   int            `yaml:"headerDelay"`
	FooterDelay   int            `yaml:"footerDelay"`
	CharDelay     int            `yaml:"charDelay"`
	LineDelay     int            `yaml:"lineDelay"`
	BlockDelay    int            `yaml:"blockDelay"`
	WaitCondition *WaitCondition `yaml:"waitCondition"`
	WaitCallBack  func(chan bool)
}

type WaitCondition struct {
	Commands []string `yaml:"commands"`
	Delay    int      `yaml:"delay"`
}

func (s *Script) sender() {
	for _, block := range s.Blocks {
		if block.Header != "" {
			textSender(block.Header, block.CharDelay, block.HeaderDelay)
		}
		textSender(block.Text, block.CharDelay, block.LineDelay)
		if block.WaitCondition != nil {
			var waitChan = make(chan bool)
			go executeWaitCondition(block.WaitCondition, waitChan)
			<-waitChan
		}
		time.Sleep(time.Duration(block.LineDelay) * time.Millisecond)
		if block.Footer != "" {
			textSender(block.Footer, block.CharDelay, block.FooterDelay)
		}
	}
}

func executeWaitCondition(waitCondition *WaitCondition, waitChan chan bool) {
	fmt.Println("da")
	var waitConditionMap = make(map[int]bool)
	for {
		for idx, command := range waitCondition.Commands {
			cmdList := strings.Split(command, " ")
			cmd := exec.Command(cmdList[0], cmdList[1:]...)
			fmt.Printf("executing condition command %d %s\n", idx, command)
			if err := cmd.Run(); err != nil {
				fmt.Printf("command %s execution failed, waitiong %d seconds, err: %s\n", command, waitCondition.Delay, err)
				time.Sleep(time.Second * time.Duration(waitCondition.Delay))
				continue
			}
			fmt.Printf("command %s executed\n", command)
			waitConditionMap[idx] = true
			allReady := true
			for idx2, _ := range waitCondition.Commands {
				if _, ok := waitConditionMap[idx2]; !ok {
					allReady = false
					break
				}
			}
			if allReady {
				waitChan <- true
			}
		}
	}

}

func textSender(text string, charDelay, lineDelay int) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		for _, r := range scanner.Text() {
			sendRune(r)
			time.Sleep(time.Duration(charDelay) * time.Millisecond)
		}
		fmt.Println(scanner.Text())
		sendEnter()
		time.Sleep(time.Duration(lineDelay) * time.Millisecond)
	}
}

func sendRune(r rune) {
	sendChar := fmt.Sprintf("%c", r)
	if sendChar == ";" {
		sendChar = sendChar + ";"
	}
	cmdList := []string{"/usr/local/bin/tmux", "send-keys", "-l", "-t", "0", sendChar}
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(err, out.String())
	}
}

func sendEnter() {
	cmdList := []string{"/usr/local/bin/tmux", "send-keys", "-t", "0", "ENTER"}
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(err, out.String())
	}
}
