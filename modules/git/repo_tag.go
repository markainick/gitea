// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"strings"

	"github.com/mcuadros/go-version"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// TagPrefix tags prefix path on the repository
const TagPrefix = "refs/tags/"

// IsTagExist returns true if given tag exists in the repository.
func IsTagExist(repoPath, name string) bool {
	return IsReferenceExist(repoPath, TagPrefix+name)
}

// IsTagExist returns true if given tag exists in the repository.
func (repo *Repository) IsTagExist(name string) bool {
	_, err := repo.gogitRepo.Reference(plumbing.ReferenceName(TagPrefix+name), true)
	if err != nil {
		return false
	}
	return true
}

// CreateTag create one tag in the repository
func (repo *Repository) CreateTag(name, revision string) error {
	_, err := NewCommand("tag", name, revision).RunInDir(repo.Path)
	return err
}

func (repo *Repository) getTag(id SHA1) (*Tag, error) {
	t, ok := repo.tagCache.Get(id.String())
	if ok {
		log("Hit cache: %s", id)
		return t.(*Tag), nil
	}

	// Get tag type
	tp, err := NewCommand("cat-file", "-t", id.String()).RunInDir(repo.Path)
	if err != nil {
		return nil, err
	}
	tp = strings.TrimSpace(tp)

	// Tag is a commit.
	if ObjectType(tp) == ObjectCommit {
		tag := &Tag{
			ID:     id,
			Object: id,
			Type:   string(ObjectCommit),
			repo:   repo,
		}

		repo.tagCache.Set(id.String(), tag)
		return tag, nil
	}

	// Tag with message.
	data, err := NewCommand("cat-file", "-p", id.String()).RunInDirBytes(repo.Path)
	if err != nil {
		return nil, err
	}

	tag, err := parseTagData(data)
	if err != nil {
		return nil, err
	}

	tag.ID = id
	tag.repo = repo

	repo.tagCache.Set(id.String(), tag)
	return tag, nil
}

// GetTag returns a Git tag by given name.
func (repo *Repository) GetTag(name string) (*Tag, error) {
	idStr, err := repo.GetTagCommitID(name)
	if err != nil {
		return nil, err
	}

	id, err := NewIDFromString(idStr)
	if err != nil {
		return nil, err
	}

	tag, err := repo.getTag(id)
	if err != nil {
		return nil, err
	}
	tag.Name = name
	return tag, nil
}

// GetTagInfos returns all tag infos of the repository.
func (repo *Repository) GetTagInfos() ([]*Tag, error) {
	// TODO this a slow implementation, makes one git command per tag
	stdout, err := NewCommand("tag").RunInDir(repo.Path)
	if err != nil {
		return nil, err
	}

	tagNames := strings.Split(stdout, "\n")
	var tags = make([]*Tag, 0, len(tagNames))
	for _, tagName := range tagNames {
		tagName = strings.TrimSpace(tagName)
		if len(tagName) == 0 {
			continue
		}

		tag, err := repo.GetTag(tagName)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	sortTagsByTime(tags)
	return tags, nil
}

// GetTags returns all tags of the repository.
func (repo *Repository) GetTags() ([]string, error) {
	var tagNames []string

	tags, err := repo.gogitRepo.Tags()
	if err != nil {
		return nil, err
	}

	tags.ForEach(func(tag *plumbing.Reference) error {
		tagNames = append(tagNames, strings.TrimPrefix(tag.Name().String(), TagPrefix))
		return nil
	})

	version.Sort(tagNames)

	// Reverse order
	for i := 0; i < len(tagNames)/2; i++ {
		j := len(tagNames) - i - 1
		tagNames[i], tagNames[j] = tagNames[j], tagNames[i]
	}

	return tagNames, nil
}
