package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/blang/semver"
	"github.com/srizzling/aquatic/version"
)

type GitBranch struct {
	Name string
}

type GitCommit struct {
	ShortHash string
	LongHash  string
}

type GitTag struct {
	Major  string
	Minor  string
	Patch  string
	Raw    string
	SemVer bool
}

type DockerTag struct {
}

var (
	versionFlag bool
	image       string
	imagePath   string
)

const banner = `
aquatic - tag docker images with git metadata
Version: %s
GitCommitSHA: %s
`

func init() {
	flag.StringVar(&image, "img", "", "The image name of the docker image")
	flag.StringVar(&imagePath, "imgPath", ".", "The path to the docker image (defaults to: .)")
	flag.BoolVar(&versionFlag, "v", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(banner, version.Version, version.GitCommitSHA))
		flag.PrintDefaults()
	}

	flag.Parse()

	if versionFlag {
		fmt.Printf(banner, version.Version, version.GitCommitSHA)
		os.Exit(0)
	}

	if image == "" {
		usageAndExit("Image name cannot be empty", 1)
	}
}

func main() {
	tag, err := getTag()
	if err != nil {
		panic(err)
	}

	commit, err := getCommit()
	if err != nil {
		panic(err)
	}

	branch, err := getBranch()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Tag: %s.%s.%s\n", tag.Major, tag.Minor, tag.Patch)
	fmt.Printf("Branch: %s\n", branch.Name)
	fmt.Printf("Commit: long:%s|short:%s\n", commit.LongHash, commit.ShortHash)
}

func runGit(args ...string) (string, error) {
	var cmd = exec.Command("git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", errors.New(stderr.String())
	}
	return stdout.String(), nil
}

// getTag tries to imitate `git describe --tags` command to retreive the tag on the HEAD
func getTag() (*GitTag, error) {
	raw, err := runGit("describe", "--tags", "--abbrev=0")
	if err != nil {
		return nil, err
	}
	tag := strings.TrimSpace(raw)

	// Check if tag is semver compliant
	// does the tag start with v? strip it since it not actually semver complaint
	if strings.HasPrefix(tag, "v") {
		// strip the v from the tag
		tag = tag[1:]
	}

	v, err := semver.Make(tag)
	if err != nil {
		// well the tag isn't semver compliant.. so lets just return the raw value
		return &GitTag{
			Raw:    tag,
			SemVer: true,
		}, nil
	}

	// unfourently git describe doesn't return a semver compliant tag
	// so lets just move it to build information
	return &GitTag{
		Major:  fmt.Sprint(v.Major),
		Minor:  fmt.Sprint(v.Minor),
		Patch:  fmt.Sprint(v.Patch),
		Raw:    tag,
		SemVer: true,
	}, nil
}

func getCommit() (*GitCommit, error) {
	longHash, err := runGit("rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}

	shortHash, err := runGit("rev-parse", "--short", "HEAD")
	if err != nil {
		return nil, err
	}

	return &GitCommit{
		LongHash:  strings.TrimSpace(longHash),
		ShortHash: strings.TrimSpace(shortHash),
	}, nil
}

func getBranch() (*GitBranch, error) {
	name, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	return &GitBranch{
		Name: strings.TrimSpace(name),
	}, nil
}

func usageAndExit(message string, exitCode int) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(exitCode)
}
