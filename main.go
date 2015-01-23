package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"github.com/masterzen/winrm/winrm"
	"github.com/packer-community/winrmcp/winrmcp"
)

var usage string = `
Usage: winrmls [options] [-help | <dir>]

  List files in a remote directory.

Options:

  -addr=localhost:5985  Host and port of the remote machine
  -user=""              Name of the user to authenticate as
  -pass=""              Password to authenticate with

`

func main() {
	if hasSwitch("-help") {
		fmt.Print(usage)
		os.Exit(0)
	}
	if err := runMain(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func runMain() (err error) {

	flags := flag.NewFlagSet("cli", flag.ContinueOnError)
	flags.Usage = func() { fmt.Print(usage) }
	user := flags.String("user", "vagrant", "winrm admin username")
	pass := flags.String("pass", "vagrant", "winrm admin password")
	addr := flags.String("addr", "localhost:5985", "host address and port")
	flags.Parse(os.Args[1:])

	args := flags.Args()
	if len(args) != 1 {
		return errors.New("The name of a remote directory is required")
	}

	endpoint, err := parseEndpoint(*addr)
	if err != nil {
		return errors.New(fmt.Sprintf("Couldn't parse addr: %v\n", err))
	}

	client := winrm.NewClient(endpoint, *user, *pass)
	err = uploadScript(client)
	if err != nil {
		return errors.New(fmt.Sprintf("Couldn't upload script: %v\n", err))
	}
	err = executeScript(client, args[0])
	if err != nil {
		return errors.New(fmt.Sprintf("Couldn't execute ls script: %v\n", err))
	}

	return
}

func uploadScript(client *winrm.Client) error {
	// grab script
	data, err := Asset("posh/List.ps1")
	if err != nil {
		return err
	}

	// copy script
	buffer := bytes.NewBuffer(data)
	cp := winrmcp.New(client)

	return cp.Write("C:/Windows/Temp/List.ps1", buffer)
}

func executeScript(client *winrm.Client, remotePath string) error {
	shell, err := client.CreateShell()

	if err != nil {
		errors.New(fmt.Sprintf("Couldn't create shell: %v", err))
	}

	defer shell.Close()
	scriptPath := "C:/Windows/Temp/List.ps1"
	cmd, err := shell.Execute("powershell", "-File "+scriptPath, "-RemotePath "+remotePath)
	if err != nil {
		return errors.New(fmt.Sprintf("Couldn't execute script %s: %v", scriptPath, err))
	}

	defer cmd.Close()
	go io.Copy(os.Stdout, cmd.Stdout)
	go io.Copy(os.Stderr, cmd.Stderr)
	cmd.Wait()

	if cmd.ExitCode() != 0 {
		return errors.New(fmt.Sprintf("List operation returned bad code: %d", cmd.ExitCode()))
	}

	return nil
}

func hasSwitch(name string) bool {
	for _, arg := range os.Args[1:] {
		if arg == name {
			return true
		}
	}
	return false
}

func parseEndpoint(addr string) (*winrm.Endpoint, error) {
	var iport int
	host, sport, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	if sport == "" {
		iport = 5985
	} else {
		iport, err = strconv.Atoi(sport)
		if err != nil {
			return nil, err
		}
	}
	return &winrm.Endpoint{
		Host: host, Port: iport,
	}, nil
}
