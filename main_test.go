package main

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"testing"
)

func captureSender(t *testing.T) *notificationRequest {
	t.Helper()
	original := sendNotification
	t.Cleanup(func() { sendNotification = original })

	request := new(notificationRequest)
	sendNotification = func(value notificationRequest) error {
		*request = value
		return nil
	}
	return request
}

func TestRunMiseInvocation(t *testing.T) {
	request := captureSender(t)
	var stdout, stderr bytes.Buffer

	code := run([]string{
		"--app-name", "mise",
		"--urgency", "normal",
		"--icon", `/mnt/c/cache/notifications/mise.png`,
		"--",
		"Conflict detected", "Run mise dot status",
	}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run() exit code = %d, stderr = %q", code, stderr.String())
	}
	if request.Summary != "Conflict detected" || request.Body != "Run mise dot status" {
		t.Fatalf("notification text = %#v", request)
	}
	if request.Icon != `/mnt/c/cache/notifications/mise.png` || request.AppName != "mise" || request.Urgency != "normal" {
		t.Fatalf("notification options = %#v", request)
	}
}

func TestRunSupportsLibnotifyOptions(t *testing.T) {
	request := captureSender(t)

	code := run([]string{
		"-u", "critical",
		"-t", "15000",
		"-a", "example",
		"-i", "icon.png",
		"-n", "app-icon.png",
		"-c", "device,transfer",
		"-e",
		"-h", "string:desktop-entry:example",
		"--hint", "boolean:resident:true",
		"-p",
		"--id-fd=-1",
		"-r", "42",
		"-w",
		"-A", "open=Open",
		"--action", "dismiss=Dismiss",
		"--selected-action-fd=-1",
		"--activation-token-fd=-1",
		"Title", `line\nbody`,
	}, &bytes.Buffer{}, &bytes.Buffer{})

	if code != 0 {
		t.Fatalf("run() exit code = %d", code)
	}
	if request.Urgency != "critical" || request.ExpireTime != 15000 || request.AppName != "example" {
		t.Fatalf("basic options = %#v", request)
	}
	if request.Icon != "icon.png" || request.AppIcon != "app-icon.png" || request.Category != "device,transfer" {
		t.Fatalf("icon/category options = %#v", request)
	}
	if !request.Transient || !request.PrintID || !request.Wait {
		t.Fatalf("boolean options = %#v", request)
	}
	if request.ReplaceID != 42 || len(request.Hints) != 2 || len(request.Actions) != 2 {
		t.Fatalf("repeatable/options = %#v", request)
	}
	if request.Body != "line\nbody" {
		t.Fatalf("body = %q", request.Body)
	}
}

func TestRunShortHelpDoesNotConflictWithHint(t *testing.T) {
	request := captureSender(t)
	code := run([]string{"-h", "string:desktop-entry:mise", "Title"}, &bytes.Buffer{}, &bytes.Buffer{})

	if code != 0 || len(request.Hints) != 1 || request.Hints[0] != "string:desktop-entry:mise" {
		t.Fatalf("exit code = %d, request = %#v", code, request)
	}
}

func TestRunHelp(t *testing.T) {
	var stdout bytes.Buffer
	code := run([]string{"-?"}, &stdout, &bytes.Buffer{})

	if code != 0 || !strings.Contains(stdout.String(), "Usage: notify-send") {
		t.Fatalf("exit code = %d, stdout = %q", code, stdout.String())
	}
}

func TestRunRejectsInvalidUrgency(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"--urgency", "urgent", "Title"}, &bytes.Buffer{}, &stderr)

	if code != 2 || !strings.Contains(stderr.String(), "invalid urgency") {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunRejectsWrongArgumentCount(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"one", "two", "three"}, &bytes.Buffer{}, &stderr)

	if code != 2 || !strings.Contains(stderr.String(), "expected <summary> [body]") {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunReturnsBackendFailure(t *testing.T) {
	original := sendNotification
	t.Cleanup(func() { sendNotification = original })
	sendNotification = func(notificationRequest) error {
		return errors.New("toast unavailable")
	}

	var stderr bytes.Buffer
	code := run([]string{"Title"}, &bytes.Buffer{}, &stderr)

	if code != 3 || !strings.Contains(stderr.String(), "toast unavailable") {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunPrintsNotificationID(t *testing.T) {
	captureSender(t)
	var stdout bytes.Buffer
	code := run([]string{"--print-id", "Title"}, &stdout, &bytes.Buffer{})

	if code != 0 {
		t.Fatalf("run() exit code = %d", code)
	}
	if _, err := strconv.Atoi(strings.TrimSpace(stdout.String())); err != nil {
		t.Fatalf("stdout = %q: %v", stdout.String(), err)
	}
}

func TestRunVersion(t *testing.T) {
	var stdout bytes.Buffer
	code := run([]string{"--version"}, &stdout, &bytes.Buffer{})

	if code != 0 || !strings.Contains(stdout.String(), "notify-send "+Version) {
		t.Fatalf("exit code = %d, stdout = %q", code, stdout.String())
	}
}
