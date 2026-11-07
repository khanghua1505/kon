package cmd

// ExitError carries the exit code kubie has to terminate with.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return ""
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// ExitCode reports the process exit code.
func (e *ExitError) ExitCode() int {
	return e.Code
}
