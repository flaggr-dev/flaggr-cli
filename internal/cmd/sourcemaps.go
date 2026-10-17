package cmd

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/flaggr-dev/flaggr-cli/internal/api"
	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

// maxSourceMapBytes is the largest map the API takes, decompressed.
const maxSourceMapBytes = 64 << 20

func newSourceMapsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sourcemaps",
		Short: "Upload source maps, so issues show minified stack traces as your source",
		Long: `Upload, list and delete a project's source maps. Flaggr uses them to show
the frames of minified stack traces (browser bundles, bundled servers) as
the original file, line and source on the Issues page.

Upload the maps of each release you deploy, with the release your app reports
(the SDK's release option):

  flaggr sourcemaps upload .next/static --url-prefix /_next/static --release "$GIT_SHA"

Each map is filed under the path of the minified file it belongs to, as stack
traces show it: --url-prefix plus the map's path in the directory, without
".map". A project keeps the maps of its last 20 releases.`,
	}
	cmd.AddCommand(newSourceMapsUploadCmd(), newSourceMapsListCmd(), newSourceMapsDeleteCmd())
	return cmd
}

func projectOrDefault(projectID string) (string, error) {
	if projectID != "" {
		return projectID, nil
	}
	if cfg != nil && cfg.DefaultProjectID != "" {
		return cfg.DefaultProjectID, nil
	}
	return "", errors.New("--project is required (no default project is saved)")
}

// sourceMapTarget is one map to upload and the minified file it is for.
type sourceMapTarget struct {
	Path string // on disk
	File string // in stack traces
}

// findSourceMaps lists the .map files under each argument (a directory, or
// a .map file), each with the path of its minified file: urlPrefix plus its
// path relative to the directory, without ".map".
func findSourceMaps(args []string, urlPrefix string) ([]sourceMapTarget, error) {
	prefix := "/" + strings.Trim(urlPrefix, "/")
	if prefix == "/" {
		prefix = ""
	}
	var targets []sourceMapTarget
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if !strings.HasSuffix(arg, ".map") {
				return nil, fmt.Errorf("%s is not a .map file", arg)
			}
			targets = append(targets, sourceMapTarget{Path: arg, File: prefix + "/" + strings.TrimSuffix(filepath.Base(arg), ".map")})
			continue
		}
		err = filepath.WalkDir(arg, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (d.Name() == "node_modules" || strings.HasPrefix(d.Name(), ".") && p != arg) {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".map") {
				return nil
			}
			rel, err := filepath.Rel(arg, p)
			if err != nil {
				return err
			}
			targets = append(targets, sourceMapTarget{Path: p, File: prefix + "/" + strings.TrimSuffix(path.Clean(filepath.ToSlash(rel)), ".map")})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].File < targets[j].File })
	return targets, nil
}

func gzipFile(p string) ([]byte, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxSourceMapBytes {
		return nil, fmt.Errorf("%s is over 64 MiB", p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func newSourceMapsUploadCmd() *cobra.Command {
	var projectID, release, urlPrefix string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "upload <dir|file.map>...",
		Short: "Upload the .map files under directories (or single .map files)",
		Args:  cobra.MinimumNArgs(1),
		// A failed upload is reported per file above; usage text would bury it.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := projectOrDefault(projectID)
			if err != nil {
				return err
			}
			targets, err := findSourceMaps(args, urlPrefix)
			if err != nil {
				return err
			}
			if len(targets) == 0 {
				return errors.New("no .map files found")
			}
			if dryRun {
				for _, t := range targets {
					fmt.Printf("%s → %s\n", t.Path, t.File)
				}
				return nil
			}

			type result struct {
				target sourceMapTarget
				upload *api.SourceMapUpload
				err    error
			}
			results := make([]result, len(targets))
			jobs := make(chan int)
			var wg sync.WaitGroup
			for w := 0; w < 4; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						t := targets[i]
						body, err := gzipFile(t.Path)
						if err == nil {
							results[i].upload, err = rest.UploadSourceMap(cmd.Context(), project, release, t.File, body)
						}
						results[i].target, results[i].err = t, err
					}
				}()
			}
			for i := range targets {
				jobs <- i
			}
			close(jobs)
			wg.Wait()

			failed := 0
			for _, r := range results {
				if r.err != nil {
					failed++
					output.Error(fmt.Sprintf("%s: %v", r.target.File, r.err))
				}
			}
			if jsonOut {
				out := make([]map[string]any, 0, len(results))
				for _, r := range results {
					entry := map[string]any{"path": r.target.Path, "file": r.target.File}
					if r.err != nil {
						entry["error"] = r.err.Error()
					} else {
						entry["bytes"], entry["replaced"] = r.upload.Bytes, r.upload.Replaced
					}
					out = append(out, entry)
				}
				output.JSON(out)
			} else {
				label := release
				if label == "" {
					label = "(no release)"
				}
				output.Success(fmt.Sprintf("Uploaded %d of %d source maps for %s", len(targets)-failed, len(targets), label))
			}
			if failed > 0 {
				return fmt.Errorf("%d uploads failed", failed)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (default: the saved default project)")
	cmd.Flags().StringVarP(&release, "release", "r", "", "The release these maps belong to, as your app reports it")
	cmd.Flags().StringVar(&urlPrefix, "url-prefix", "", "Path the directory is served under, such as /_next/static")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "List the maps and the files they'd be filed under, without uploading")
	return cmd
}

func newSourceMapsListCmd() *cobra.Command {
	var projectID, release string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List releases with source maps, or one release's files",
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := projectOrDefault(projectID)
			if err != nil {
				return err
			}
			if cmd.Flags().Changed("release") {
				files, err := rest.ListSourceMapFiles(cmd.Context(), project, release)
				if err != nil {
					return err
				}
				if jsonOut {
					output.JSON(files)
					return nil
				}
				rows := make([][]string, len(files))
				for i, f := range files {
					rows[i] = []string{f.File, strconv.FormatInt(f.Bytes, 10), f.UploadedAt}
				}
				output.Table([]string{"FILE", "BYTES", "UPLOADED"}, rows)
				return nil
			}
			releases, err := rest.ListSourceMapReleases(cmd.Context(), project)
			if err != nil {
				return err
			}
			if jsonOut {
				output.JSON(releases)
				return nil
			}
			rows := make([][]string, len(releases))
			for i, r := range releases {
				name := r.Release
				if name == "" {
					name = "(no release)"
				}
				rows[i] = []string{name, strconv.Itoa(r.Files), strconv.FormatInt(r.Bytes, 10), r.LastUploadedAt}
			}
			output.Table([]string{"RELEASE", "FILES", "BYTES", "LAST UPLOAD"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (default: the saved default project)")
	cmd.Flags().StringVarP(&release, "release", "r", "", "List this release's files")
	return cmd
}

func newSourceMapsDeleteCmd() *cobra.Command {
	var projectID, release, file string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a release's source maps (or one file's)",
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := projectOrDefault(projectID)
			if err != nil {
				return err
			}
			if !cmd.Flags().Changed("release") {
				return errors.New(`--release is required (use --release "" for maps uploaded without one)`)
			}
			deleted, err := rest.DeleteSourceMaps(cmd.Context(), project, release, file)
			if err != nil {
				return err
			}
			if jsonOut {
				output.JSON(map[string]int{"deleted": deleted})
				return nil
			}
			output.Success(fmt.Sprintf("Deleted %d source maps", deleted))
			return nil
		},
	}
	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (default: the saved default project)")
	cmd.Flags().StringVarP(&release, "release", "r", "", "The release whose maps to delete")
	cmd.Flags().StringVar(&file, "file", "", "Delete only this file's map")
	return cmd
}
