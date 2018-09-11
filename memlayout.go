package memlayout

import (
	"context"

	log "gopkg.in/src-d/go-log.v1"
	lookout "gopkg.in/src-d/lookout-sdk.v0/pb"
)

// Analyzer of memory layout.
type Analyzer struct {
	version    string
	dataServer string
}

// NewAnalyzer creates a new memlayout analyzer.
func NewAnalyzer(version, dataServer string) *Analyzer {
	return &Analyzer{version: version, dataServer: dataServer}
}

// NotifyReviewEvent implements the lookout analyzer interface.
func (a *Analyzer) NotifyReviewEvent(ctx context.Context, review *lookout.ReviewEvent) (*lookout.EventResponse, error) {
	log.Infof("got review request %v", review)
	return &lookout.EventResponse{AnalyzerVersion: a.version, Comments: nil}, nil
}

// NotifyPushEvent implements the lookout analyzer interface.
func (*Analyzer) NotifyPushEvent(context.Context, *lookout.PushEvent) (*lookout.EventResponse, error) {
	return &lookout.EventResponse{}, nil
}
