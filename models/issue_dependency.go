// Copyright 2018 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"context"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/timeutil"
)

// IssueDependency represents an issue dependency
type IssueDependency struct {
	ID           int64              `xorm:"pk autoincr"`
	UserID       int64              `xorm:"NOT NULL"`
	IssueID      int64              `xorm:"UNIQUE(issue_dependency) NOT NULL"`
	DependencyID int64              `xorm:"UNIQUE(issue_dependency) NOT NULL"`
	CreatedUnix  timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix  timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(IssueDependency))
}

// DependencyType Defines Dependency Type Constants
type DependencyType int

// Define Dependency Types
const (
	DependencyTypeBlockedBy DependencyType = iota
	DependencyTypeBlocking
)

// CreateIssueDependency creates a new dependency for an issue
func CreateIssueDependency(user *user_model.User, issue, dep *Issue) error {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()
	sess := db.GetEngine(ctx)

	// Check if it aleready exists
	exists, err := issueDepExists(sess, issue.ID, dep.ID)
	if err != nil {
		return err
	}
	if exists {
		return ErrDependencyExists{issue.ID, dep.ID}
	}
	// And if it would be circular
	circular, err := issueDepExists(sess, dep.ID, issue.ID)
	if err != nil {
		return err
	}
	if circular {
		return ErrCircularDependency{issue.ID, dep.ID}
	}

	if err := db.Insert(ctx, &IssueDependency{
		UserID:       user.ID,
		IssueID:      issue.ID,
		DependencyID: dep.ID,
	}); err != nil {
		return err
	}

	// Add comment referencing the new dependency
	if err = createIssueDependencyComment(ctx, user, issue, dep, true); err != nil {
		return err
	}

	return committer.Commit()
}

// RemoveIssueDependency removes a dependency from an issue
func RemoveIssueDependency(user *user_model.User, issue, dep *Issue, depType DependencyType) (err error) {
	ctx, committer, err := db.TxContext()
	if err != nil {
		return err
	}
	defer committer.Close()

	var issueDepToDelete IssueDependency

	switch depType {
	case DependencyTypeBlockedBy:
		issueDepToDelete = IssueDependency{IssueID: issue.ID, DependencyID: dep.ID}
	case DependencyTypeBlocking:
		issueDepToDelete = IssueDependency{IssueID: dep.ID, DependencyID: issue.ID}
	default:
		return ErrUnknownDependencyType{depType}
	}

	affected, err := db.GetEngine(ctx).Delete(&issueDepToDelete)
	if err != nil {
		return err
	}

	// If we deleted nothing, the dependency did not exist
	if affected <= 0 {
		return ErrDependencyNotExists{issue.ID, dep.ID}
	}

	// Add comment referencing the removed dependency
	if err = createIssueDependencyComment(ctx, user, issue, dep, false); err != nil {
		return err
	}
	return committer.Commit()
}

// Check if the dependency already exists
func issueDepExists(e db.Engine, issueID, depID int64) (bool, error) {
	return e.Where("(issue_id = ? AND dependency_id = ?)", issueID, depID).Exist(&IssueDependency{})
}

// IssueNoDependenciesLeft checks if issue can be closed
func IssueNoDependenciesLeft(ctx context.Context, issue *Issue) (bool, error) {
	exists, err := db.GetEngine(ctx).
		Table("issue_dependency").
		Select("issue.*").
		Join("INNER", "issue", "issue.id = issue_dependency.dependency_id").
		Where("issue_dependency.issue_id = ?", issue.ID).
		And("issue.is_closed = ?", "0").
		Exist(&Issue{})

	return !exists, err
}
