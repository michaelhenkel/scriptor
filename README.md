# Introduction

Scriptor allows to define a sequence of commands to be sent to a tmux session and execute them.   
Speed of keystrokes, delays and waiting conditions between the commands can be defined.

# Example

## Define the sequence
```bash
cat << EOF > script1.yaml
tmuxSession: s1
blocks:
- command: |
    until cat /tmp/bla; do echo command1 is waiting; sleep 2; done
    sleep 10
    touch /tmp/bla2
  header: |
    #First command (slow typer)
  footer: |
    #First command done
  blockDelay: 2000
  charDelay: 500 
  lineDelay: 1000
  headerDelay: 1200
  clear: true
  tmuxPane: 0
- command: |
    sleep 20
    touch /tmp/bla
    until cat /tmp/bla2; do echo command2 is waiting; sleep 2; done
    touch /tmp/bla3
  header: |
    #Second command (fast typer)
  waitCondition:
    commands:
    - ls /tmp/bla3
    delay: 3
  blockDelay: 6000
  charDelay: 50
  lineDelay: 1000
  headerDelay: 1200
  footerDelay: 1200
  clear: true
  tmuxPane: 1
EOF
```

## Create a tmux session
```bash
tmux new -s s1 -d
tmux split-window -h
tmux attach -t s1
```

## Run
In a different terminal:   
```bash
go run main.go script1.yaml
```