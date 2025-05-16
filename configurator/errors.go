package configurator

import "fmt"

type InvalidConfigError struct {
	Message string
}

func (e *InvalidConfigError) Error() string {
	return fmt.Sprintf("invalid configuration: %s", e.Message)
}
