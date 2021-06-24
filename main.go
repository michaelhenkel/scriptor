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
	TMUXSession string  `yaml:"tmuxSession"`
	Blocks      []Block `yaml:"blocks"`
}

type Block struct {
	Clear         bool           `yaml:"clear"`
	Command       string         `yaml:"command"`
	Header        string         `yaml:"header"`
	Footer        string         `yaml:"fooder"`
	HeaderDelay   int            `yaml:"headerDelay"`
	FooterDelay   int            `yaml:"footerDelay"`
	CharDelay     int            `yaml:"charDelay"`
	LineDelay     int            `yaml:"lineDelay"`
	BlockDelay    int            `yaml:"blockDelay"`
	WaitCondition *WaitCondition `yaml:"waitCondition"`
	WaitCallBack  func(chan bool)
	TMUXPane      string `yaml:"tmuxPane"`
}

type WaitCondition struct {
	Commands []string `yaml:"commands"`
	Delay    int      `yaml:"delay"`
}

func (s *Script) sender() {
	if s.TMUXSession == "" {
		s.TMUXSession = "0"
	}
	for _, block := range s.Blocks {
		if block.TMUXPane == "" {
			block.TMUXPane = "0"
		}
		tmuxSessionPane := fmt.Sprintf("%s.%s", s.TMUXSession, block.TMUXPane)
		if block.Clear {
			textSender("clear", 0, 0, tmuxSessionPane)
		}
		if block.Header != "" {
			textSender(block.Header, 0, block.HeaderDelay, tmuxSessionPane)
		}
		textSender(block.Command, block.CharDelay, block.LineDelay, tmuxSessionPane)
		if block.WaitCondition != nil {
			var waitChan = make(chan bool)
			go executeWaitCondition(block.WaitCondition, waitChan)
			<-waitChan
		}
		time.Sleep(time.Duration(block.BlockDelay) * time.Millisecond)
		if block.Footer != "" {
			textSender(block.Footer, block.CharDelay, block.FooterDelay, tmuxSessionPane)
		}
	}
}

func executeWaitCondition(waitCondition *WaitCondition, waitChan chan bool) {
	var waitConditionMap = make(map[int]bool)
	for {
		for idx, command := range waitCondition.Commands {
			cmd := exec.Command("sh", "-c", command)
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

func textSender(text string, charDelay, lineDelay int, tmuxSessionPane string) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		for _, r := range scanner.Text() {
			sendRune(r, tmuxSessionPane)
			time.Sleep(time.Duration(charDelay) * time.Millisecond)
		}
		fmt.Println(scanner.Text())
		sendEnter(tmuxSessionPane)
		time.Sleep(time.Duration(lineDelay) * time.Millisecond)
	}
}

func sendRune(r rune, tmuxSessionPane string) {
	sendChar := fmt.Sprintf("%c", r)
	if sendChar == ";" {
		sendChar = sendChar + ";"
	}
	cmdList := []string{"tmux", "send-keys", "-l", "-t", tmuxSessionPane, sendChar}
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(err, out.String())
	}
}

func sendEnter(tmuxSessionPane string) {
	cmdList := []string{"tmux", "send-keys", "-t", tmuxSessionPane, "ENTER"}
	cmd := exec.Command(cmdList[0], cmdList[1:]...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		fmt.Println(err, out.String())
	}
}
