package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

func main() {
	script := Script{
		Blocks: []Block{{
			BlockDelay:  1300,
			CharDelay:   5,
			LineDelay:   1000,
			HeaderDelay: 1200,
			Header:      `#Set cluster count, memory and cpu`,
			Text: `clusterNodes=(0 1)
memory=8g
cpu=4

`,
		}, {BlockDelay: 1000,
			CharDelay:    5,
			LineDelay:    10,
			HeaderDelay:  1200,
			WaitCallBack: waitForMinikube,
			Header:       `#Run minikube and contrail`,
			Text: `deployerLocation=~/deployer.yaml
for cluster in ${clusterNodes}; do
  sed "/metadata:/{n;s/name: contrail-k8s-kubemanager/name: c${cluster}/;}" ${deployerLocation} > deployer_c${cluster}.yaml
  sed -i "s/autonomousSystem: 64512/autonomousSystem: 6451${cluster}/g" deployer_c${cluster}.yaml
  minikube start -p c${cluster} --driver hyperkit --cni ~/deployer_c${cluster}.yaml --container-runtime crio --memory ${memory} --cpus ${cpu} &
done

`,
		}, {BlockDelay: 1000,
			CharDelay:    5,
			LineDelay:    10,
			HeaderDelay:  1200,
			WaitCallBack: waitForControlnodes,
			Header: `

#Wait for control node to be up`,
			Text: `for cluster in ${clusterNodes}; do
  kubectl config use-context c${cluster}
  until kubectl -n contrail get pod contrail-control-0; do sleep 5; done
done

`,
		}, {BlockDelay: 1000,
			CharDelay:   5,
			LineDelay:   10,
			HeaderDelay: 1200,
			Header: `

#Switch to first cluster`,
			Text: `kubectl config use-context c${clusterNodes[@]:0:1}

`,
		}, {BlockDelay: 1000,
			CharDelay:   5,
			LineDelay:   10,
			HeaderDelay: 1200,
			//WaitCallBack: waitForControlnodes,
			Header: `

#Federate controlplane`,
			Text: `contrail_federate create

`,
		}, {BlockDelay: 1000,
			CharDelay:   5,
			LineDelay:   10,
			HeaderDelay: 1200,
			//WaitCallBack: waitForControlnodes,
			Header: `

#Add kubefed helm charts`,
			Text: `helm repo add kubefed-charts https://raw.githubusercontent.com/kubernetes-sigs/kubefed/master/charts
ver=$(helm search repo kubefed -ojson | jq ".[0].version" |tr -d "\"")
helm --namespace kube-federation-system upgrade -i kubefed kubefed-charts/kubefed --version=${ver} --create-namespace

`,
		}},
	}
	script.sender()
}

type Script struct {
	Blocks []Block
}

type Block struct {
	Text         string
	Header       string
	Footer       string
	HeaderDelay  int
	CharDelay    int
	LineDelay    int
	BlockDelay   int
	WaitCallBack func(chan bool)
}

func (s *Script) sender() {
	for _, block := range s.Blocks {
		if block.Header != "" {
			textSender(block.Header, block.CharDelay, block.HeaderDelay)
		}
		textSender(block.Text, block.CharDelay, block.LineDelay)
		if block.WaitCallBack != nil {
			var waitChan = make(chan bool)
			go block.WaitCallBack(waitChan)
			<-waitChan
		}
		time.Sleep(time.Duration(block.LineDelay) * time.Millisecond)
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

func waitForMinikube(waitChan chan bool) {
	var minikubeRunningMap = make(map[string]bool)
	for {
		var out bytes.Buffer
		minikubes := []string{"c0", "c1"}
		for _, minikube := range minikubes {
			cmd := exec.Command("minikube", "-p", minikube, "status")
			cmd.Stdout = &out
			if err := cmd.Run(); err != nil {
				fmt.Println(err, out.String())
			}
			r, _ := regexp.Compile(fmt.Sprintf(`%s
type: Control Plane
host: Running
kubelet: Running
apiserver: Running
kubeconfig: Configured`, minikube))
			running := r.FindString(out.String())
			if running != "" {
				minikubeRunningMap[minikube] = true
				allRunning := true
				for _, mk := range minikubes {
					if _, ok := minikubeRunningMap[mk]; !ok {
						allRunning = false
						break
					}
				}
				if allRunning {
					fmt.Println("all minikubes are runing")
					waitChan <- true
				}
			}
			fmt.Printf("minikube %s not running, waiting\n", minikube)
			time.Sleep(2 * time.Second)
		}
	}
}

func waitForControlnodes(waitChan chan bool) {
	var minikubeRunningMap = make(map[string]bool)
	for {
		var out bytes.Buffer
		minikubes := []string{"c0", "c1"}
		for _, minikube := range minikubes {
			cmd := exec.Command("kubectl", "-n", "contrail", "get", "pods", "contrail-control-0", "--context", minikube)
			cmd.Stdout = &out
			if err := cmd.Run(); err != nil {
				fmt.Println(err, out.String())
				fmt.Printf("control-nodes on %s not running, waiting\n", minikube)
				time.Sleep(2 * time.Second)
			} else {
				minikubeRunningMap[minikube] = true
				allRunning := true
				for _, mk := range minikubes {
					if _, ok := minikubeRunningMap[mk]; !ok {
						allRunning = false
						break
					}
				}
				if allRunning {
					fmt.Println("all minikubes are runing")
					waitChan <- true
				}
			}

		}
	}
}
