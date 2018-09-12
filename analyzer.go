package memlayout

import (
	"context"
	"fmt"
	"io"

	"github.com/davecgh/go-spew/spew"
	"google.golang.org/grpc"
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

	conn, err := grpc.Dial(a.dataServer, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DataServer at %s", a.dataServer)
	}
	defer conn.Close()

	dataClient := lookout.NewDataClient(conn)
	changes, err := dataClient.GetChanges(ctx, &lookout.ChangesRequest{
		Head:            &review.Head,
		Base:            &review.Base,
		WantContents:    true,
		WantUAST:        false,
		ExcludeVendored: true,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting changes from data server %s", a.dataServer)
	}

	var comments []*lookout.Comment
	for {
		change, err := changes.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not receive changes from data server %s", a.dataServer)
		}

		log.Infof("analyzing '%s' in %s", change.Head.Path, change.Head.Language)

		structs, err := StructsFromSource(change.Head.Path, change.Head.Content)
		if err != nil {
			log.Warningf("unable to type check file, it's not implemented right now")
			continue
		}

		comments = append(comments, &lookout.Comment{
			File: change.Head.Path,
			Line: 0,
			Text: fmt.Sprintf("%d structs", len(structs)),
		})
	}

	spew.Dump(review)
	return &lookout.EventResponse{AnalyzerVersion: a.version, Comments: nil}, nil
}

// NotifyPushEvent implements the lookout analyzer interface.
func (*Analyzer) NotifyPushEvent(context.Context, *lookout.PushEvent) (*lookout.EventResponse, error) {
	return &lookout.EventResponse{}, nil
}
