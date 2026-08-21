package platform

import "log"

func Shutdown(gracePeriod int) error {
	log.Printf(
		"SHUTDOWN WOULD BE TRIGGERED (grace period: %d seconds)",
		gracePeriod,
	)

	return nil
}

func CancelShutdown() error {
	log.Println("SHUTDOWN CANCEL WOULD BE TRIGGERED")

	return nil
}
