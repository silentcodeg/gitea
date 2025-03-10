// Copyright 2021 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package attachment

import (
	"os"
	"path/filepath"
	"testing"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
	})
}

func TestUploadAttachment(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)

	fPath := "./attachment_test.go"
	f, err := os.Open(fPath)
	assert.NoError(t, err)
	defer f.Close()

	attach, err := NewAttachment(&repo_model.Attachment{
		RepoID:     1,
		UploaderID: user.ID,
		Name:       filepath.Base(fPath),
	}, f)
	assert.NoError(t, err)

	attachment, err := repo_model.GetAttachmentByUUID(attach.UUID)
	assert.NoError(t, err)
	assert.EqualValues(t, user.ID, attachment.UploaderID)
	assert.Equal(t, int64(0), attachment.DownloadCount)
}
