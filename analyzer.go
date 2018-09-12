package memlayout

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

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

	repoPath, err := clone(review)
	if err != nil {
		return nil, fmt.Errorf("unable to clone repo: %s", err)
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

		comments = append(comments, commentsForChanges(repoPath, change)...)
	}

	return &lookout.EventResponse{
		AnalyzerVersion: a.version,
		Comments:        comments,
	}, nil
}

// NotifyPushEvent implements the lookout analyzer interface.
func (*Analyzer) NotifyPushEvent(context.Context, *lookout.PushEvent) (*lookout.EventResponse, error) {
	return &lookout.EventResponse{}, nil
}

func commentsForChanges(repoPath string, change *lookout.Change) []*lookout.Comment {
	if change.Head == nil {
		return nil
	}

	log.Infof("analyzing %q", change.Head.Path)

	headStructs, err := StructsFromFile(filepath.Join(repoPath, change.Head.Path), change.Head.Content)
	if err != nil {
		log.Errorf(err, "unable to get structs from head revision")
		return nil
	}

	var structNames []string
	for _, s := range headStructs {
		structNames = append(structNames, s.Name)
	}

	log.Debugf("structs found in HEAD: %s", strings.Join(structNames, ", "))

	var base []byte
	if change.Base != nil {
		base = change.Base.Content
	}

	changed := ChangedStructs(
		base, change.Head.Content,
		headStructs,
	)

	structNames = make([]string, 0, len(changed))
	for _, s := range changed {
		structNames = append(structNames, s.Name)
	}

	log.Debugf("these structs changed: %s", strings.Join(structNames, ", "))

	var result []*lookout.Comment
	for _, c := range changed {
		optimized := Optimize(c)
		log.Debugf("for struct %q padding was %d, but could be optimized to %d", c.Name, c.Padding(), optimized.Padding())
		if optimized.Padding() >= c.Padding() {
			continue
		}

		var comment = &lookout.Comment{
			Line: int32(c.Start),
			File: change.Head.Path,
		}

		var buf bytes.Buffer
		buf.WriteString("We've detected the memory layout could be improved to reduce padding.")
		buf.WriteString("\n\n**Struct info:**\n")
		for _, f := range c.Fields {
			buf.WriteString(f.String())
			buf.WriteRune('\n')
		}
		buf.WriteString(fmt.Sprintf("\nHere's the proposed layout:\n\n```go\n%s\n```", optimized))
		log.Debugf("comment was added with suggestions for struct %s", c.Name)
		comment.Text = buf.String()
		result = append(result, comment)
	}

	return result
}
