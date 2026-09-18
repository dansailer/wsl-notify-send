package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

var Version = "0.1.0"

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type notificationRequest struct {
	Summary           string
	Body              string
	AppName           string
	Icon              string
	AppIcon           string
	Urgency           string
	Category          string
	ExpireTime        int
	Transient         bool
	PrintID           bool
	IDFD              int
	ReplaceID         int
	Wait              bool
	Actions           stringList
	SelectedActionFD  int
	ActivationTokenFD int
	Hints             stringList
}

type options struct {
	notificationRequest
	debug   bool
	version bool
	help    bool
}

var sendNotification = sendWindowsNotification

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, positional, err := parseOptions(args)
	if errors.Is(err, flag.ErrHelp) {
		printUsage(stdout)
		return 0
	}
	if err != nil {
		writef(stderr, "notify-send: %v\n", err)
		return 2
	}

	if opts.help {
		printUsage(stdout)
		return 0
	}
	if opts.version {
		writef(stdout, "notify-send %s\n", Version)
		return 0
	}
	if len(positional) < 1 {
		writeLine(stderr, "notify-send: requires a summary argument")
		return 2
	}
	if len(positional) > 2 {
		writeLine(stderr, "notify-send: expected <summary> [body]")
		return 2
	}

	request := opts.notificationRequest
	request.Summary = positional[0]
	if len(positional) == 2 {
		request.Body = unescapeBody(positional[1])
	}
	if request.AppIcon != "" && request.Icon == "" {
		request.Icon = request.AppIcon
	}
	if len(request.Actions) > 0 {
		request.Wait = true
	}

	if err := sendNotification(request); err != nil {
		writef(stderr, "notify-send: %v\n", err)
		return 3
	}

	if request.PrintID || request.IDFD >= 0 {
		id := newNotificationID()
		if request.PrintID {
			writeLine(stdout, id)
		}
		if request.IDFD >= 0 {
			if err := writeNotificationID(request.IDFD, id); err != nil {
				writef(stderr, "notify-send: could not write notification ID: %v\n", err)
				return 3
			}
		}
	}

	if opts.debug {
		writef(stderr, "notify-send: notification accepted by Windows backend (app=%q urgency=%s)\n", request.AppName, request.Urgency)
	}

	return 0
}

func parseOptions(args []string) (options, []string, error) {
	opts := options{
		notificationRequest: notificationRequest{
			AppName:           "notify-send",
			Urgency:           "normal",
			ExpireTime:        -1,
			IDFD:              -1,
			ReplaceID:         0,
			SelectedActionFD:  -1,
			ActivationTokenFD: -1,
		},
	}

	flags := flag.NewFlagSet("notify-send", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&opts.AppName, "app-name", opts.AppName, "application name")
	flags.StringVar(&opts.AppName, "a", opts.AppName, "application name")
	flags.StringVar(&opts.Icon, "icon", "", "icon filename or stock icon")
	flags.StringVar(&opts.Icon, "i", "", "icon filename or stock icon")
	flags.StringVar(&opts.AppIcon, "app-icon", "", "application icon filename or name")
	flags.StringVar(&opts.AppIcon, "n", "", "application icon filename or name")
	flags.StringVar(&opts.Urgency, "urgency", opts.Urgency, "urgency: low, normal, or critical")
	flags.StringVar(&opts.Urgency, "u", opts.Urgency, "urgency: low, normal, or critical")
	flags.IntVar(&opts.ExpireTime, "expire-time", opts.ExpireTime, "expiration timeout in milliseconds")
	flags.IntVar(&opts.ExpireTime, "t", opts.ExpireTime, "expiration timeout in milliseconds")
	flags.StringVar(&opts.Category, "category", "", "notification category")
	flags.StringVar(&opts.Category, "c", "", "notification category")
	flags.BoolVar(&opts.Transient, "transient", false, "create a transient notification")
	flags.BoolVar(&opts.Transient, "e", false, "create a transient notification")
	flags.Var(&opts.Hints, "hint", "hint in TYPE:NAME:VALUE format")
	flags.Var(&opts.Hints, "h", "hint in TYPE:NAME:VALUE format")
	flags.BoolVar(&opts.PrintID, "print-id", false, "print the notification ID")
	flags.BoolVar(&opts.PrintID, "p", false, "print the notification ID")
	flags.IntVar(&opts.IDFD, "id-fd", opts.IDFD, "file descriptor for the notification ID")
	flags.IntVar(&opts.ReplaceID, "replace-id", opts.ReplaceID, "notification ID to replace")
	flags.IntVar(&opts.ReplaceID, "r", opts.ReplaceID, "notification ID to replace")
	flags.BoolVar(&opts.Wait, "wait", false, "wait for the notification to close")
	flags.BoolVar(&opts.Wait, "w", false, "wait for the notification to close")
	flags.Var(&opts.Actions, "action", "notification action")
	flags.Var(&opts.Actions, "A", "notification action")
	flags.IntVar(&opts.SelectedActionFD, "selected-action-fd", opts.SelectedActionFD, "file descriptor for the selected action")
	flags.IntVar(&opts.ActivationTokenFD, "activation-token-fd", opts.ActivationTokenFD, "file descriptor for the activation token")
	flags.BoolVar(&opts.version, "version", false, "show version")
	flags.BoolVar(&opts.version, "v", false, "show version")
	flags.BoolVar(&opts.help, "help", false, "show help")
	flags.BoolVar(&opts.help, "?", false, "show help")
	flags.BoolVar(&opts.debug, "debug", false, "print backend diagnostics")

	if err := flags.Parse(args); err != nil {
		return options{}, nil, err
	}

	opts.Urgency = strings.ToLower(strings.TrimSpace(opts.Urgency))
	switch opts.Urgency {
	case "low", "normal", "critical":
	default:
		return options{}, nil, fmt.Errorf("invalid urgency %q (expected low, normal, or critical)", opts.Urgency)
	}

	return opts, flags.Args(), nil
}

func unescapeBody(body string) string {
	if !strings.Contains(body, `\`) {
		return body
	}

	quoted := `"` + strings.ReplaceAll(body, `"`, `\"`) + `"`
	decoded, err := strconv.Unquote(quoted)
	if err != nil {
		return body
	}
	return decoded
}

func newNotificationID() int {
	return int(time.Now().UnixNano() & 0x7fffffff)
}

func writeNotificationID(fd, id int) error {
	file := os.NewFile(uintptr(fd), "notification-id")
	if file == nil {
		return fmt.Errorf("invalid file descriptor %d", fd)
	}
	_, writeErr := fmt.Fprintf(file, "%d\n", id)
	return errors.Join(writeErr, file.Close())
}

func printUsage(w io.Writer) {
	writeLine(w, "Usage: notify-send [OPTION...] <SUMMARY> [BODY]")
	writeLine(w)
	writeLine(w, "Options:")
	writeLine(w, "  -u, --urgency=LEVEL")
	writeLine(w, "  -t, --expire-time=TIME")
	writeLine(w, "  -a, --app-name=APP_NAME")
	writeLine(w, "  -i, --icon=ICON")
	writeLine(w, "  -n, --app-icon=ICON")
	writeLine(w, "  -c, --category=CATEGORY")
	writeLine(w, "  -e, --transient")
	writeLine(w, "  -h, --hint=TYPE:NAME:VALUE")
	writeLine(w, "  -p, --print-id")
	writeLine(w, "      --id-fd=FD")
	writeLine(w, "  -r, --replace-id=ID")
	writeLine(w, "  -w, --wait")
	writeLine(w, "  -A, --action=[NAME=]TEXT")
	writeLine(w, "      --selected-action-fd=FD")
	writeLine(w, "      --activation-token-fd=FD")
	writeLine(w, "  -?, --help")
	writeLine(w, "  -v, --version")
	writeLine(w, "      --debug")
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeLine(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}
