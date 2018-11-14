package main

import (
	"os"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/libgit2/git2go.v25"

	"vccfinder/ds"
)

type Commit struct {
	gitCommit      *git.Commit
	gitRepository  *git.Repository
	Sha            string
}

func main() {
	fmt.Println("Starting...")
	log.SetLevel(log.DebugLevel)

	repoPath := "repos/hub"
	fixSHA := "016ec99d25b1cb83cb4367e541177aa431beb600"
	commit := new(Commit)
	commit.Sha = fixSHA
	repo, err := git.OpenRepository(repoPath)
	if err != nil {
		log.Errorf("Failed to open repository")
		os.Exit(1)
	}
	commit.gitRepository = repo

	commitOid, err := git.NewOid(commit.Sha)
	if err != nil {
		log.Errorf("Failed to get oid")
		os.Exit(1)
	}
	gitCommit, err := commit.gitRepository.LookupCommit(commitOid)
	if err != nil {
		log.Errorf("Failed to get commit")
		os.Exit(1)
	}

	commit.gitCommit = gitCommit

	blamed, err := commit.getBlameCommitSha()
	if err != nil {
		log.Errorf("Failed to get blame sha")
		log.Fatal(err)
		os.Exit(1)
	}
	fmt.Println("Blamed: " + blamed)
}

func (c *Commit) getBlameCommitSha() (blamedCommit string, err error) {
	blamedCommits := ds.NewMaxMap()
	repo := c.gitRepository

	diff, parent, err := c.diff()
	if err != nil {
		return
	}
	defer diff.Free()

	err = diff.ForEach(func(delta git.DiffDelta, num float64) (git.DiffForEachHunkCallback, error) {
		var blame *Blame

		log.Debugf("%v: %s is code -> %v", c, delta.OldFile.Path, IsCodeFile(delta.OldFile.Path))
		if delta.Status != git.DeltaAdded && IsCodeFile(delta.OldFile.Path) {
			blame, err = NewBlame(repo, parent.Id().String(), delta.OldFile.Path, BlameBackward)
		}

		additionBlock := false
		return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
			return func(line git.DiffLine) error {
				// only consider deleted lines
				if blame == nil {
					return nil
				}
				var lineToBlame int
				switch {
				case line.Origin == git.DiffLineAddition:
					if !additionBlock {
						// first added line in addition block -> blame previous line
						additionBlock = true
						lineToBlame = line.NewLineno - 1
					}
				case line.Origin == git.DiffLineDeletion || additionBlock:
					// Blame on deleted lines OR if addition block ended
					additionBlock = false
					lineToBlame = line.OldLineno
				}
				if lineToBlame > 0 {
					bl, err := blame.ForLine(lineToBlame)
					if err != nil {
						log.Errorf("%v: could not get blame for line %d", c, lineToBlame)
						return nil
					}
					log.Debugf("%v: blame line %d -> %s", c, lineToBlame, bl.Sha)
					blamedCommits.Add(bl.Sha)
				}

				return nil
			}, nil
		}, err
	}, git.DiffDetailLines)

	blamedCommit, _ = blamedCommits.MaxString()
	if blamedCommit == "" {
		err = fmt.Errorf("no blamed commit found (%+v)", blamedCommits)
	} else {
		log.Infof("%s: blame %s", c.String(), blamedCommit)
	}

	return
}

func (c *Commit) String() string {
	return fmt.Sprintf("%s", c.Sha)
}

func (c *Commit) diff() (diff *git.Diff, parent *git.Commit, err error) {
	var pTree *git.Tree
	gitCommit, err := c.GitCommit()
	if err != nil {
		return
	}
	repo:= c.gitRepository
	diffOpts, _ := git.DefaultDiffOptions()
	diffOpts.Flags = git.DiffIgnoreFilemode
	parent = gitCommit.Parent(0)
	if parent == nil {
		//return nil, nil, fmt.Errorf("Initial commit")
		pTree = new(git.Tree) // use empty tree
	} else {
		pTree, err = parent.Tree()
		if err != nil {
			return
		}
	}
	defer pTree.Free()
	cTree, err := gitCommit.Tree()
	defer cTree.Free()
	if err != nil {
		return
	}
	diff, err = repo.DiffTreeToTree(pTree, cTree, &diffOpts)

	return
}

// GitCommit returns the git commit for the commit (requires that the repository has been cloned)
func (c *Commit) GitCommit() (commit *git.Commit, err error) {
	if c.gitCommit != nil {
		return c.gitCommit, nil
	}

	oid, err := git.NewOid(c.Sha)
	if err != nil {
		return
	}
	repo := c.gitRepository
	commit, err = repo.LookupCommit(oid)
	if err != nil {
		return
	}
	c.gitCommit = commit

	return
}
