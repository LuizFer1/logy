package update

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Options configures a self-update Run.
type Options struct {
	CurrentVersion string
	GOOS, GOARCH   string
	ExePath        string // os.Executable()
	Getter         Getter
	Repo           string
	Yes            bool
	CheckOnly      bool
	Stdin          io.Reader
	Stdout         io.Writer
	Prompt         func(question string) (bool, error) // default: read Y/n from Stdin
}

// Result summarizes what Run decided or did.
type Result struct {
	Current string
	Latest  string
	Updated bool
	Pending bool // true when CheckOnly && update available
}

// Run checks GitHub Releases and optionally replaces the local executable.
func Run(opts Options) (Result, error) {
	out := opts.Stdout
	if out == nil {
		out = io.Discard
	}
	in := opts.Stdin
	if in == nil {
		in = os.Stdin
	}
	g := opts.Getter
	if g == nil {
		g = HTTPGetter{}
	}
	repo := opts.Repo
	if repo == "" {
		repo = RepoSlug()
	}
	prompt := opts.Prompt
	if prompt == nil {
		prompt = func(question string) (bool, error) {
			fmt.Fprint(out, question)
			line, err := bufio.NewReader(in).ReadString('\n')
			if err != nil && err != io.EOF {
				return false, err
			}
			s := strings.TrimSpace(line)
			if s == "" || strings.EqualFold(s, "y") || strings.EqualFold(s, "yes") {
				return true, nil
			}
			return false, nil
		}
	}

	res := Result{Current: opts.CurrentVersion}

	rel, err := LatestRelease(g, repo)
	if err != nil {
		return res, err
	}
	res.Latest = rel.TagName

	if CompareSemver(opts.CurrentVersion, rel.TagName) >= 0 {
		fmt.Fprintf(out, "Already up to date (%s)\n", rel.TagName)
		return res, nil
	}

	fmt.Fprintf(out, "Current: %s\n", opts.CurrentVersion)
	fmt.Fprintf(out, "Latest:  %s\n", rel.TagName)

	if opts.CheckOnly {
		res.Pending = true
		return res, nil
	}

	if !opts.Yes {
		ok, err := prompt(fmt.Sprintf("Update to %s? [Y/n] ", rel.TagName))
		if err != nil {
			return res, err
		}
		if !ok {
			return res, nil
		}
	}

	assetName, err := AssetName(rel.TagName, opts.GOOS, opts.GOARCH)
	if err != nil {
		return res, err
	}
	assetURL, err := FindAssetURL(rel, assetName)
	if err != nil {
		return res, err
	}
	checksumURL, err := FindChecksumURL(rel)
	if err != nil {
		return res, err
	}

	fmt.Fprintf(out, "Downloading %s …\n", assetName)
	checksumBody, err := g.Get(checksumURL)
	if err != nil {
		return res, fmt.Errorf("download checksums: %w", err)
	}
	sums, err := ParseChecksums(string(checksumBody))
	if err != nil {
		return res, err
	}
	want, ok := sums[assetName]
	if !ok {
		return res, fmt.Errorf("checksum missing for %s", assetName)
	}

	archive, err := g.Get(assetURL)
	if err != nil {
		return res, fmt.Errorf("download archive: %w", err)
	}
	if err := VerifySHA256(archive, want); err != nil {
		return res, err
	}
	fmt.Fprintln(out, "Checksum OK")

	if opts.ExePath == "" {
		return res, fmt.Errorf("ExePath is required")
	}
	newPath := opts.ExePath + ".new"
	if err := ExtractBinary(archive, assetName, newPath); err != nil {
		_ = os.Remove(newPath)
		return res, err
	}
	if err := ReplaceExecutable(opts.ExePath, newPath); err != nil {
		_ = os.Remove(newPath)
		return res, err
	}

	fmt.Fprintf(out, "Updated successfully to %s\n", rel.TagName)
	res.Updated = true
	return res, nil
}
