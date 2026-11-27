package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.ReadCloser, stdout io.Writer, stderr io.Writer) error {
	flags := flag.NewFlagSet("go-telnet", flag.ContinueOnError)
	flags.SetOutput(stderr)

	timeout := 10 * time.Second
	flags.DurationVar(&timeout, "timeout", timeout, "connection timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	if flags.NArg() != 2 {
		return errors.New("usage: go-telnet [--timeout=10s] host port")
	}

	host := flags.Arg(0)
	portText := flags.Arg(1)
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: %q", portText)
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))
	client := NewTelnetClient(address, timeout, stdin, stdout)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("connection to %s failed: %w", address, err)
	}
	fmt.Fprintf(stderr, "...Connected to %s\n", address)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sendDone := make(chan error, 1)
	receiveDone := make(chan error, 1)

	go func() {
		sendDone <- client.Send()
	}()
	go func() {
		receiveDone <- client.Receive()
	}()

	var locallyClosed atomic.Bool
	closeLocal := func() {
		locallyClosed.Store(true)
		_ = client.Close()
		_ = stdin.Close()
	}

	select {
	case <-ctx.Done():
		closeLocal()
		return nil
	case err := <-sendDone:
		closeLocal()
		if err != nil {
			return fmt.Errorf("send failed: %w", err)
		}
		fmt.Fprintln(stderr, "...EOF")
		return nil
	case err := <-receiveDone:
		if locallyClosed.Load() {
			return nil
		}
		closeLocal()
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("receive failed: %w", err)
		}
		fmt.Fprintln(stderr, "...Connection was closed by peer")
		return nil
	}
}
