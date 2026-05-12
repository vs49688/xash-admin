package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/adrg/xdg"
	"github.com/chzyer/readline"
	"github.com/google/shlex"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"git.vs49688.net/zane/xash-admin/config"
)

const AppName = "xash-admin, a Xash3D-FWGS RCON CLI"

func displayHeader(w io.Writer) {
	_, _ = fmt.Fprintf(w, `%s %s - Copyright (C) 2026 Zane van Iperen
  Contact: zane@zanevaniperen.com

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License version 2, and only
version 2 as published by the Free Software Foundation.
`, AppName, config.AppVersion)
}

func pcCVarEnum(name string, values ...string) readline.PrefixCompleterInterface {
	pcItems := make([]readline.PrefixCompleterInterface, len(values))
	for i, item := range values {
		pcItems[i] = readline.PcItem(item)
	}

	return readline.PcItem(name, pcItems...)
}

func pcCVar01(name string) readline.PrefixCompleterInterface {
	return pcCVarEnum(name, "0", "1")
}

var completer = readline.NewPrefixCompleter(
	//
	// System completions
	//
	readline.PcItem("!!exit"),
	readline.PcItem("!exit"),
	readline.PcItem("!setpassword"),
	readline.PcItem("!sleep"),

	//
	// Server completions
	//
	readline.PcItem("changelevel"),
	readline.PcItem("map"),
	pcCVarEnum("maps", "*"),
	readline.PcItem("status"),
	readline.PcItem("serverinfo"),
	readline.PcItem("mapcyclefile"),

	pcCVar01("sv_aim"),
	pcCVar01("sv_allowdownload"),
	pcCVar01("sv_allowupload"),
	pcCVar01("sv_busters"),
	pcCVar01("sv_cheats"),
	pcCVarEnum("sv_gravity", "800"),         // 800 = Default
	pcCVarEnum("sv_maxspeed", "270", "320"), // 270 = MP default, 320 = SP default
	pcCVarEnum("sv_stepsize", "18"),         //  18 = Default

	pcCVar01("mp_autocrosshair"),
	pcCVar01("mp_allowmonsters"),
	pcCVar01("mp_falldamage"),
	readline.PcItem("mp_fraglimit"),
	pcCVar01("mp_friendlyfire"),
	pcCVar01("mp_flashlight"),
	pcCVar01("mp_footsteps"),
	pcCVar01("mp_forcerespawn"),
	readline.PcItem("mp_timeleft"),
	readline.PcItem("mp_timelimit"),
	pcCVar01("mp_weaponstay"),

	readline.PcItem("hostname"),
	readline.PcItem("maxplayers"),
	pcCVar01("pausable"),
	pcCVar01("teamplay"),

	readline.PcItem("exec"),
	readline.PcItem("allow_spectators"),
	readline.PcItem("rcon_password"),

	//
	// Mod Completions
	//

	// metamod
	readline.PcItem("meta",
		readline.PcItem("version"),
		readline.PcItem("game"),
		readline.PcItem("list"),
		readline.PcItem("cmds"),
		readline.PcItem("cvars"),
		readline.PcItem("refresh"),
		readline.PcItem("config"),
		readline.PcItem("load"),
		readline.PcItem("unload"),
		readline.PcItem("reload"),
		readline.PcItem("info"),
		readline.PcItem("pause"),
		readline.PcItem("unpause"),
		readline.PcItem("retry"),
		readline.PcItem("clear"),
		readline.PcItem("force_unload"),
		readline.PcItem("require"),
	),

	// jk_botti
	readline.PcItem("jk_botti",
		pcCVar01("show_waypoints"),
	),
	pcCVarEnum("jk_botti_trace", "0", "1", "2"),
	readline.PcItem("jk_botti_version"),

	// hlmetrics
	readline.PcItem("hlmetrics_port"),
)

func buildRCONMessage(password, command string) []byte {
	return []byte(fmt.Sprintf("\xFF\xFF\xFF\xFFrcon %s %s\x00", strconv.QuoteToASCII(password), command))
}

type Message struct {
	Time    time.Time
	Message []byte
	Error   error
	IsWrite bool
}

var quakeColorReplacer = strings.NewReplacer(
	"^0", "\033[30m", // black
	"^1", "\033[31m", // red
	"^2", "\033[32m", // green
	"^3", "\033[33m", // yellow
	"^4", "\033[34m", // blue
	"^5", "\033[36m", // cyan
	"^6", "\033[35m", // magenta
	"^7", "\033[0m", // default/reset
)

var quakeColorStripper = strings.NewReplacer(
	"^0", "", // black
	"^1", "", // red
	"^2", "", // green
	"^3", "", // yellow
	"^4", "", // blue
	"^5", "", // cyan
	"^6", "", // magenta
	"^7", "", // default/reset
)

func netReader(conn net.Conn, ch chan<- Message, getNow func() time.Time) {
	var a2cPrintPrefix = []byte{0xFF, 0xFF, 0xFF, 0xFF, 'p', 'r', 'i', 'n', 't', '\n'}

	for {
		var buf = make([]byte, 4096)
		n, err := conn.Read(buf)
		if errors.Is(err, os.ErrDeadlineExceeded) {
			// Our main thread uses conn.SetReadDeadline() to kill us.
			break
		} else if err != nil {
			ch <- Message{Time: getNow(), Error: err}
			continue
		}

		if !bytes.HasPrefix(buf[:n], a2cPrintPrefix) {
			continue
		}

		ch <- Message{
			Time:    getNow(),
			Message: buf[len(a2cPrintPrefix):n],
		}
	}
}

func netWriter(conn net.Conn, ch <-chan []byte, msgChan chan<- Message, getNow func() time.Time) {
	for packet := range ch {
		if _, err := conn.Write(packet); err != nil {
			msgChan <- Message{Time: getNow(), Error: err, IsWrite: true}
		}
	}
}

func repl(l *readline.Instance, password string, cmdChan chan<- []byte) {
	setPasswordCfg := l.GenPasswordConfig()
	setPasswordCfg.SetListener(func(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
		l.SetPrompt(fmt.Sprintf("Enter password: %s", strings.Repeat("*", len(line))))
		l.Refresh()
		return nil, 0, false
	})

	for {
		line, err := l.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		} else if errors.Is(err, io.EOF) {
			break
		}

		line = strings.TrimSpace(line)
		argv, _ := shlex.Split(line)

		switch {
		case len(argv) == 1 && argv[0] == "!setpassword":
			newPass, err := l.ReadPasswordWithConfig(setPasswordCfg)
			if err != nil {
				fmt.Printf("Error setting password: %s\n", err)
				continue
			}
			password = string(newPass)

		case len(argv) == 2 && argv[0] == "!setpassword":
			password = argv[1]

		case len(argv) > 2 && argv[0] == "!setpassword":
			fmt.Printf("usage: !setpassword [password]\n")

		case len(argv) > 0 && argv[0] == "exit":
			_, _ = fmt.Fprintf(l.Stdout(), "Refusing %s, use %s to quit client, %s to quit server\n",
				strconv.Quote("exit"),
				strconv.Quote("!exit"),
				strconv.Quote("!!exit"),
			)

		case len(argv) > 0 && argv[0] == "!exit":
			return

		case len(argv) > 0 && argv[0] == "!!exit":
			cmdChan <- buildRCONMessage(password, "exit")

		case len(argv) > 0 && argv[0] == "!sleep":
			switch len(argv) {
			case 2:
				if duration, err := time.ParseDuration(strings.TrimSpace(argv[1])); err == nil {
					<-time.After(duration)
					break
				}

				fallthrough
			default:
				_, _ = fmt.Fprintf(l.Stdout(), "usage: !sleep <duration> (e.g. 1s, 5m, 1h)\n")
			}

		case line == "":
		default:
			msgBytes := buildRCONMessage(password, line)
			cmdChan <- msgBytes
		}
	}
}

func forEachLine(msg *Message, stripColours bool, proc func(line string)) {
	lines := strings.Split(strings.TrimRightFunc(string(msg.Message), unicode.IsSpace), "\n")
	for _, line := range lines {
		line = strings.TrimRightFunc(line, unicode.IsSpace)

		if stripColours {
			line = quakeColorStripper.Replace(line)
		} else {
			line = quakeColorReplacer.Replace(line)
		}

		proc(line)
	}
}

func run(ctx context.Context, command *cli.Command) error {
	isTerminal := term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd())) //#nosec G115

	address := command.String("address")
	password := command.String("password")
	historyFile := command.String("history-file")

	if !command.IsSet("history-file") {
		hf, err := xdg.DataFile("xash-admin/history.tmp")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "unable to create default history file, history will not be saved: %s\n", err.Error())
		}
		historyFile = hf
	}

	if isTerminal {
		displayHeader(os.Stdout)
		fmt.Printf("\nServer Address: %s\n", strconv.Quote(address))

		if password == "" {
			fmt.Printf("Password:       (unset)\n")
		} else {
			fmt.Printf("Password:       %s\n", strings.Repeat("*", len(password)))
		}

		if historyFile == "" {
			fmt.Printf("History File:   \"\" (disabled)\n")
		} else {
			fmt.Printf("History File:   %s\n", strconv.Quote(historyFile))
		}

		fmt.Printf("\n")

		if password == "" {
			fmt.Printf("WARNING: Password not set, please manually specify using !setpassword\n")
		}

		fmt.Printf("\n")
	}

	l, err := readline.NewEx(&readline.Config{
		Prompt:                 "\u001B[31m(rcon) $\u001B[0m ",
		HistoryFile:            historyFile,
		DisableAutoSaveHistory: false,
		AutoComplete:           completer,
		InterruptPrompt:        "^C",
		EOFPrompt:              "!!exit",
		HistorySearchFold:      true,
		FuncIsTerminal:         func() bool { return isTerminal },
		FuncFilterInputRune: func(r rune) (rune, bool) {
			switch r {
			// block CtrlZ feature
			case readline.CharCtrlZ:
				return r, false
			}
			return r, true
		},
	})
	if err != nil {
		return fmt.Errorf("error initialising readline: %w", err)
	}
	defer func() { _ = l.Close() }()

	conn, err := net.Dial("udp", address)
	if err != nil {
		return fmt.Errorf("error dialing address: %w", err)
	}
	defer func() { _ = conn.Close() }()

	l.CaptureExitSignal()

	msgChan := make(chan Message, 1024)
	cmdChan := make(chan []byte, 1024)

	readerDone := make(chan struct{})
	go func() {
		netReader(conn, msgChan, time.Now)
		close(readerDone)
	}()

	writerDone := make(chan struct{})
	go func() {
		netWriter(conn, cmdChan, msgChan, time.Now)
		close(writerDone)
	}()

	replDone := make(chan struct{})
	go func() {
		repl(l, password, cmdChan)
		close(replDone)
	}()

	rlOut := l.Stdout()
	rlErr := l.Stderr()

	printTimestamps := false // FIXME: Maybe we'll need this in the future

	printLine := func(w io.Writer, level string, ts time.Time, line string) {
		if !isTerminal {
			_, _ = fmt.Fprintf(w, "%s\n", line)
			return
		}

		prefix := ""

		if printTimestamps {
			prefix = "[" + ts.Format(time.RFC3339) + "] "
		}

		_, _ = fmt.Fprintf(w, "%s\u001B[31m(%s)\u001B[0m %s\n", prefix, level, line)
	}

	for {
		select {
		case <-ctx.Done():
			// If the context is done, tell readline to die.
			// We'll continue through the replDone channel.
			// NB: This doesn't actually work though: https://github.com/chzyer/readline/issues/217
			_ = l.Close()

		case msg := <-msgChan:
			if msg.Error != nil {
				printLine(rlErr, "error", msg.Time, msg.Error.Error())
			} else {
				forEachLine(&msg, !isTerminal, func(line string) {
					printLine(rlOut, "server", msg.Time, line)
				})
			}
			break

		case <-replDone:
			goto done
		}
	}

done:

	close(cmdChan)
	_ = conn.SetReadDeadline(time.Unix(1, 0))
	<-writerDone
	<-readerDone
	close(msgChan)
	return nil
}

func main() {
	root := cli.Command{
		Name:  "xash-admin",
		Usage: AppName,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Usage:   "server address",
				Value:   "127.0.0.1:27015",
				Sources: cli.EnvVars("XASH_ADMIN_ADDRESS"),
			},
			&cli.StringFlag{
				Name:    "password",
				Usage:   "rcon password",
				Sources: cli.EnvVars("XASH_ADMIN_PASSWORD"),
			},
			&cli.StringFlag{
				Name:    "history-file",
				Usage:   "command history file, empty to disable, unset to use XDG default",
				Sources: cli.EnvVars("XASH_ADMIN_HISTORY_FILE"),
			},
		},
		Action: run,
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "display version information",
				Action: func(ctx context.Context, command *cli.Command) error {
					displayHeader(os.Stdout)
					return nil
				},
			},
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
