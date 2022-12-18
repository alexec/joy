package main

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	"io"
	"k8s.io/api/core/v1"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"sigs.k8s.io/yaml"
	"strings"
	"syscall"
	"time"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}

type event = any

type signalEvent struct{}

type processExitedEvent struct {
	name string
	err  error
}

const escape = "\x1b"

type state struct {
	err   error
	phase string
	msg   string
	cmd   *exec.Cmd
}

func (s *state) Write(p []byte) (n int, err error) {
	s.msg = strings.TrimSpace(string(p))
	return 0, nil
}

var states = map[string]*state{}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	in, err := os.ReadFile("dev.yaml")
	ok(err)
	pod := &v1.Pod{}
	ok(yaml.UnmarshalStrict(in, pod))

	hosts, err := os.Create("hosts")
	ok(err)
	for _, c := range pod.Spec.Containers {
		_, err := hosts.WriteString(fmt.Sprintf("%s 127.0.0.1\n", c.Name))
		ok(err)
	}
	ok(hosts.Close())

	events := make(chan event)
	for _, c := range pod.Spec.Containers {
		states[c.Name] = &state{}
	}

	go func() {
		for {
			log.Printf("%s[2J", escape)
			log.Printf("%s[H", escape)
			for _, c := range pod.Spec.Containers {
				name := c.Name
				state := states[name]
				r := map[string]string{
					"creating": "▓",
					"starting": "▓",
					"ready":    color.GreenString("▓"),
					"unready":  color.YellowString("▓"),
					"killing":  "▓",
				}[state.phase]
				m := state.msg
				if state.err != nil {
					r = color.RedString("▓")
					m = color.RedString(state.err.Error())
				}
				log.Printf("%s %s [%s] %s, %v", r, name, state.phase, m, state.err)
			}
			time.Sleep(time.Second)
		}
	}()

	for _, c := range pod.Spec.Containers {
		states[c.Name].phase = "creating"
		log, err := os.Create(filepath.Join("logs", c.Name+".log"))
		ok(err)
		defer log.Close()

		cmd := exec.Command(c.Command[0], append(c.Command[1:], c.Args...)...)
		cmd.Dir = c.WorkingDir
		cmd.Stdin = os.Stdin
		cmd.Stdout = io.MultiWriter(log, states[c.Name])
		cmd.Stderr = io.MultiWriter(log, states[c.Name])
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
		cmd.Env = os.Environ()

		for _, e := range c.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", e.Name, e.Value))
		}

		states[c.Name].cmd = cmd

		go func(name string, cmd *exec.Cmd) {
			states[name].phase = "starting"
			err := cmd.Run()
			events <- processExitedEvent{name, err}
		}(c.Name, cmd)

		if c.ReadinessProbe != nil {
			go func(name string, probe *v1.Probe) {
				initialDelay := time.Duration(probe.InitialDelaySeconds) * time.Second
				period := time.Duration(probe.PeriodSeconds) * time.Second
				if period == 0 {
					period = 10 * time.Second
				}
				time.Sleep(initialDelay)
				for {
					if httpGet := probe.HTTPGet; httpGet != nil {
						proto := strings.ToLower(string(httpGet.Scheme))
						if proto == "" {
							proto = "http"
						}
						resp, err := http.Get(fmt.Sprintf("%s://localhost:%v%s", proto, httpGet.Port.IntValue(), httpGet.Path))
						if err != nil {
							states[name].phase = "unready"
							states[name].err = err
						} else if resp.StatusCode == 200 {
							states[name].phase = "ready"
						} else {
							states[name].phase = "unready"
							states[name].err = fmt.Errorf(resp.Status)
						}
					} else {
						states[name].msg = "httpGet not supported"
					}
					time.Sleep(period)
				}
			}(c.Name, c.ReadinessProbe)
		}
	}

	go func() {
		<-ctx.Done()
		events <- signalEvent{}
	}()

	waitingFor := len(pod.Spec.Containers)

	for event := range events {
		switch obj := event.(type) {
		case signalEvent:
			for name, state := range states {
				cmd := state.cmd
				if cmd.Process != nil {
					states[name].phase = "killing"
					pgid, _ := syscall.Getpgid(cmd.Process.Pid)
					err := syscall.Kill(-pgid, syscall.SIGTERM)
					if err != nil {
						states[name].msg = err.Error()
					}
					time.Sleep(time.Second)
				}
			}
		case processExitedEvent:
			states[obj.name].phase = "exited"
			states[obj.name].err = obj.err
			waitingFor--
			if waitingFor == 0 {
				return
			}
		}
	}
}

func ok(err error) {
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
}
