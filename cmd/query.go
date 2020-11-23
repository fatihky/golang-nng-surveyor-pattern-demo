package cmd

import "time"

type query struct {
	ID               string
	Question         string
	resultch         chan queryresult
	maxExecutionTime time.Duration
}

type querymeta struct {
	timeoutAt       time.Time
	receivedResults []queryresult
}

type queryresult struct {
	ID    string
	Items []string
	err   error
}
