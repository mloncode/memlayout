package memlayout

import (
	"io/ioutil"
	"os"

	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	lookout "gopkg.in/src-d/lookout-sdk.v0/pb"
)

func clone(review *lookout.ReviewEvent) (string, error) {
	tmp, err := ioutil.TempDir(os.TempDir(), "memlayout-")
	if err != nil {
		return "", err
	}

	r, err := git.PlainClone(tmp, false, &git.CloneOptions{
		URL: review.Head.InternalRepositoryURL,
	})

	if err != nil {
		return "", err
	}
	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(review.Head.Hash),
	})
	if err != nil {
		return "", err
	}
	return tmp, nil
}
