//go:build windows

package main

import (
	"fmt"
	"time"

	"git.sr.ht/~jackmordaunt/go-toast"
)

func sendWindowsNotification(request notificationRequest) error {
	// Category, hints, actions, and D-Bus IDs are retained by the parser for
	// CLI compatibility but have no equivalent in the Windows toast API used here.
	icon := request.Icon
	if icon == "" {
		icon = request.AppIcon
	}
	stagedIcon, cleanup, err := stageIcon(icon)
	if err != nil {
		return err
	}
	defer cleanup()

	notification := toast.Notification{
		AppID: request.AppName,
		Title: request.Summary,
		Body:  request.Body,
		Icon:  stagedIcon,
	}

	if request.ExpireTime > 10000 {
		notification.Duration = toast.Long
	} else {
		notification.Duration = toast.Short
	}

	if request.Urgency == "critical" {
		notification.Audio = toast.Default
	} else {
		notification.Audio = toast.Silent
	}

	if err := notification.Push(); err != nil {
		return fmt.Errorf("could not show Windows toast: %w", err)
	}
	// Give the Windows broker time to consume the staged image before cleanup.
	time.Sleep(100 * time.Millisecond)

	return nil
}
