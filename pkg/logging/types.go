package logging

import (
	"time"
)

type crlTimeStamps struct {
	ThisUpdate     time.Time `json:"thisUpdate"`
	NextUpdate     time.Time `json:"nextUpdate"`
	NextCRLPublish time.Time `json:"nextPublish"`
}

type ErrorReport struct {
	Err         error
	Context     string
	Severity    SeverityLevel
	Criticality CriticalityLevel
}
