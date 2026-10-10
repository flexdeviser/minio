//go:build ignore
// +build ignore

// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func genLDFlags(version string) string {
	releaseTag, date := releaseTag(version)
	copyrightYear := strconv.Itoa(date.Year())
	commit := commitID()
	shortCommit := commit
	if len(shortCommit) > 12 {
		shortCommit = shortCommit[:12]
	}
	ldflagsStr := "-s -w"
	ldflagsStr += " -X github.com/minio/minio/cmd.Version=" + version
	ldflagsStr += " -X github.com/minio/minio/cmd.CopyrightYear=" + copyrightYear
	ldflagsStr += " -X github.com/minio/minio/cmd.ReleaseTag=" + releaseTag
	ldflagsStr += " -X github.com/minio/minio/cmd.CommitID=" + commit
	ldflagsStr += " -X github.com/minio/minio/cmd.ShortCommitID=" + shortCommit
	ldflagsStr += " -X github.com/minio/minio/cmd.GOPATH=" + os.Getenv("GOPATH")
	ldflagsStr += " -X github.com/minio/minio/cmd.GOROOT=" + os.Getenv("GOROOT")
	return ldflagsStr
}

// genReleaseTag prints release tag to the console for easy git tagging.
func releaseTag(version string) (string, time.Time) {
	relPrefix := "DEVELOPMENT"
	if prefix := os.Getenv("MINIO_RELEASE"); prefix != "" {
		relPrefix = prefix
	}

	relSuffix := ""
	if hotfix := os.Getenv("MINIO_HOTFIX"); hotfix != "" {
		relSuffix = hotfix
	}

	relTag := strings.Replace(version, " ", "-", -1)
	relTag = strings.Replace(relTag, ":", "-", -1)
	t, err := time.Parse("2006-01-02T15-04-05Z", relTag)
	if err != nil {
		// Fall back to a stable development tag when the version is not in
		// the standard release format (e.g. docker builds where the VERSION
		// build-arg is empty, like docs/resiliency's docker-compose).
		relTag = "GOGIT"
		t = time.Unix(0, 0).UTC()
	}
	relTag = strings.Replace(relTag, ",", "", -1)
	relTag = relPrefix + "." + relTag
	if relSuffix != "" {
		relTag += "." + relSuffix
	}

	return relTag, t
}

// commitID returns the abbreviated commit-id hash of the last commit.
func commitID() string {
	// Allow overriding from the environment, useful for build
	// environments (e.g. Docker builds) without a git checkout.
	if v := os.Getenv("MINIO_COMMIT_ID"); v != "" {
		return strings.TrimSpace(v)
	}

	// git log --format="%H" -n1
	var (
		commit []byte
		err    error
	)
	cmdName := "git"
	cmdArgs := []string{"log", "--format=%H", "-n1"}
	if commit, err = exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		// No git checkout in the build context (e.g. docker builds), fall
		// back to a placeholder instead of failing the build.
		return "unknown"
	}

	return strings.TrimSpace(string(commit))
}

func commitTime() time.Time {
	// Allow overriding from the environment, useful for build
	// environments (e.g. Docker builds) without a git checkout.
	if v := os.Getenv("MINIO_COMMIT_TIME"); v != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error parsing MINIO_COMMIT_TIME: ", err)
			os.Exit(1)
		}
		return t.UTC()
	}

	// git log --format=%cI -n1
	var (
		commitUnix []byte
		err        error
	)
	cmdName := "git"
	cmdArgs := []string{"log", "--format=%cI", "-n1"}
	if commitUnix, err = exec.Command(cmdName, cmdArgs...).Output(); err != nil {
		return time.Unix(0, 0).UTC()
	}

	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(commitUnix)))
	if err != nil {
		return time.Unix(0, 0).UTC()
	}

	return t.UTC()
}

func main() {
	var version string
	if len(os.Args) > 1 {
		version = os.Args[1]
	} else {
		version = commitTime().Format(time.RFC3339)
	}

	fmt.Println(genLDFlags(version))
}
