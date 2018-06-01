package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/template"
	"github.com/blang/semver"
	"github.com/docker/docker/client"
	"github.com/srizzling/aquatic/version"
	yaml "gopkg.in/yaml.v1"
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

type AquaticTemplate struct {
	Tag    *GitTag
	Commit *GitCommit
	Branch *GitBranch
}

type AquaticConfig struct {
	TagFormat   []string `yaml:"tag_format"`
	LabelFormat []string `yaml:"label_format"`
	ImageNames  []string `yaml:"image_names"`
}

var (
	versionFlag bool
	img         string
	imgID       string
)

const banner = `
aquatic - tag docker images with git metadata
Version: %s
GitCommitSHA: %s
`

func init() {
	flag.StringVar(&imgID, "imgID", "", "The Id of the image to tag")
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

	if imgID == "" {
		usageAndExit("Image id cannot be empty", 1)
	}
}

func main() {
	config := AquaticConfig{}
	data, err := ioutil.ReadFile(".aquatic.yml")
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	tmplData, err := getGitInfo()
	if err != nil {
		panic(err)
	}

	docker, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	for _, name := range config.ImageNames {
		err := setTag(name, tmplData, config.TagFormat, docker)
		if err != nil {
			panic(err)
		}
	}
}

func setTag(name string, tmplData *AquaticTemplate, tagFormats []string, docker *client.Client) error {
	for _, tagTemplate := range tagFormats {
		t := template.Must(template.New("tag_template").Parse(tagTemplate))
		buf := new(bytes.Buffer)
		t.Execute(buf, tmplData)
		err := docker.ImageTag(context.Background(), imgID, fmt.Sprintf("%s:%s", name, buf.String()))
		if err != nil {
			return err
		}
	}
	return nil

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

func getGitInfo() (*AquaticTemplate, error) {
	tag, err := getTag()
	if err != nil {
		return nil, err
	}

	commit, err := getCommit()
	if err != nil {
		return nil, err
	}

	branch, err := getBranch()
	if err != nil {
		return nil, err
	}

	gitTmpl := &AquaticTemplate{
		Tag:    tag,
		Branch: branch,
		Commit: commit,
	}

	return gitTmpl, nil
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
