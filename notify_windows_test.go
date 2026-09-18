//go:build windows

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"git.sr.ht/~jackmordaunt/go-toast"
)

func replaceToastPush(t *testing.T, push func(*toast.Notification) error) {
	t.Helper()
	original := pushToast
	pushToast = push
	t.Cleanup(func() { pushToast = original })
}

func TestSendWindowsNotificationMapsRequestAndCleansIcon(t *testing.T) {
	source := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(source, []byte("png test content"), 0600); err != nil {
		t.Fatal(err)
	}

	var pushed *toast.Notification
	replaceToastPush(t, func(notification *toast.Notification) error {
		pushed = notification
		if _, err := os.Stat(notification.Icon); err != nil {
			t.Fatalf("staged icon is unavailable during push: %v", err)
		}
		return nil
	})

	err := sendWindowsNotification(notificationRequest{
		AppName:    "mise",
		Summary:    "Conflict detected",
		Body:       "Run mise dot status",
		AppIcon:    source,
		Urgency:    "critical",
		ExpireTime: 15000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pushed == nil {
		t.Fatal("toast was not pushed")
	}
	if pushed.AppID != "mise" || pushed.Title != "Conflict detected" || pushed.Body != "Run mise dot status" {
		t.Fatalf("toast fields = %#v", pushed)
	}
	if pushed.Duration != toast.Long || pushed.Audio != toast.Default {
		t.Fatalf("duration/audio = %q/%q", pushed.Duration, pushed.Audio)
	}
	if pushed.Icon == source {
		t.Fatal("icon was not staged")
	}
	if _, err := os.Stat(pushed.Icon); !os.IsNotExist(err) {
		t.Fatalf("staged icon still exists: %v", err)
	}
}

func TestSendWindowsNotificationMapsDurationAndAudio(t *testing.T) {
	tests := []struct {
		name       string
		expireTime int
		urgency    string
		duration   string
		audio      string
	}{
		{name: "short normal", expireTime: 10000, urgency: "normal", duration: toast.Short, audio: toast.Silent},
		{name: "long normal", expireTime: 10001, urgency: "normal", duration: toast.Long, audio: toast.Silent},
		{name: "short critical", expireTime: 1, urgency: "critical", duration: toast.Short, audio: toast.Default},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var pushed *toast.Notification
			replaceToastPush(t, func(notification *toast.Notification) error {
				pushed = notification
				return nil
			})

			if err := sendWindowsNotification(notificationRequest{
				Summary:    "Title",
				Urgency:    test.urgency,
				ExpireTime: test.expireTime,
			}); err != nil {
				t.Fatal(err)
			}
			if pushed == nil {
				t.Fatal("toast was not pushed")
			}
			if pushed.Duration != test.duration || pushed.Audio != test.audio {
				t.Fatalf("duration/audio = %q/%q, want %q/%q", pushed.Duration, pushed.Audio, test.duration, test.audio)
			}
		})
	}
}

func TestSendWindowsNotificationPrefersExplicitIcon(t *testing.T) {
	var pushed *toast.Notification
	replaceToastPush(t, func(notification *toast.Notification) error {
		pushed = notification
		return nil
	})

	if err := sendWindowsNotification(notificationRequest{
		Summary: "Title",
		Icon:    "explicit.png",
		AppIcon: "fallback.png",
	}); err != nil {
		t.Fatal(err)
	}
	if pushed == nil {
		t.Fatal("toast was not pushed")
	}
	if pushed.Icon != "explicit.png" {
		t.Fatalf("toast icon = %q, want explicit.png", pushed.Icon)
	}
}

func TestSendWindowsNotificationWrapsPushErrorAndCleansIcon(t *testing.T) {
	source := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(source, []byte("png test content"), 0600); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("toast unavailable")
	var staged string
	replaceToastPush(t, func(notification *toast.Notification) error {
		staged = notification.Icon
		return wantErr
	})

	err := sendWindowsNotification(notificationRequest{Summary: "Title", Icon: source})
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "could not show Windows toast") {
		t.Fatalf("error = %v", err)
	}
	if staged == "" {
		t.Fatal("toast did not receive staged icon")
	}
	if _, statErr := os.Stat(staged); !os.IsNotExist(statErr) {
		t.Fatalf("staged icon still exists after push failure: %v", statErr)
	}
}
