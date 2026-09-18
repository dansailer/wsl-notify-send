//go:build !windows

package main

import "errors"

func sendWindowsNotification(notificationRequest) error {
	return errors.New("notify-send must be run as the Windows executable from WSL")
}
