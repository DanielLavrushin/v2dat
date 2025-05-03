package unpack

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/urlesistiana/v2dat/v2data"
	"go.uber.org/zap"
)

type unpackArgs struct {
	outDir  string
	print   bool
	file    string
	filters []string
}

func newGeoSiteCmd() *cobra.Command {
	args := new(unpackArgs)
	c := &cobra.Command{
		Use:   "geosite [-o output_dir] [-f tag[@attr]...]... geosite.dat",
		Args:  cobra.ExactArgs(1),
		Short: "Unpack geosite file to text files.",
		Run: func(cmd *cobra.Command, a []string) {
			args.file = a[0]
			if err := unpackGeoSite(args); err != nil {
				logger.Fatal("failed to unpack geosite", zap.Error(err))
			}
		},
		DisableFlagsInUseLine: true,
	}
	c.Flags().StringVarP(&args.outDir, "out", "o", "", "output dir")
	c.Flags().BoolVarP(&args.print, "print", "p", false, "write to stdout instead of files")
	c.Flags().StringArrayVarP(&args.filters, "filter", "f", nil, "unpack given tag and attrs")
	return c
}

func unpackGeoSite(args *unpackArgs) error {
	// shorthand
	filePath, suffixes, outDir := args.file, args.filters, args.outDir

	// “-o -”  → stream to stdout instead of writing files
	stdoutMode := outDir == "-"

	// read & decode the .dat file once
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	geoSiteList, err := v2data.LoadGeoSiteList(data)
	if err != nil {
		return err
	}

	// build tag → domains map
	entries := make(map[string][]*v2data.Domain)
	for _, gs := range geoSiteList.GetEntry() {
		tag := strings.ToLower(gs.GetCountryCode())
		entries[tag] = gs.GetDomain()
	}

	// helper that either prints or saves one tag
	save := func(suffix string, domains []*v2data.Domain) error {
		if stdoutMode {
			// header line helps when multiple tags are streamed
			fmt.Fprintf(os.Stdout, "# %s (%d domains)\n", suffix, len(domains))
			return convertV2DomainToText(domains, os.Stdout)
		}

		name := fmt.Sprintf("%s_%s.txt", fileName(filePath), suffix)
		if outDir != "" {
			name = filepath.Join(outDir, name)
		}
		logger.Info("unpacking entry",
			zap.String("tag", suffix),
			zap.Int("length", len(domains)),
			zap.String("file", name),
		)
		return convertV2DomainToTextFile(domains, name)
	}

	// specific tags requested with -f
	if len(suffixes) != 0 {
		for _, suffix := range suffixes {
			tag, attrs := splitAttrs(suffix)

			domains, ok := entries[tag]
			if !ok {
				return fmt.Errorf("cannot find entry %s", tag)
			}
			domains = filterAttrs(domains, attrs)

			if err := save(suffix, domains); err != nil {
				return fmt.Errorf("failed to save %s: %w", suffix, err)
			}
		}
		return nil
	}

	// no -f ⇒ dump everything
	for tag, domains := range entries {
		if err := save(tag, domains); err != nil {
			return fmt.Errorf("failed to save %s: %w", tag, err)
		}
	}
	return nil
}

func convertV2DomainToTextFile(domain []*v2data.Domain, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	return convertV2DomainToText(domain, f)
}

func convertV2DomainToText(domain []*v2data.Domain, w io.Writer) error {
	bw := bufio.NewWriter(w)
	for _, r := range domain {
		var prefix string
		switch r.Type {
		case v2data.Domain_Plain:
			prefix = "keyword:"
		case v2data.Domain_Regex:
			prefix = "regexp:"
		case v2data.Domain_Domain:
			prefix = ""
		case v2data.Domain_Full:
			prefix = "full:"
		default:
			return fmt.Errorf("invalid domain type %d", r.Type)
		}
		if _, err := bw.WriteString(prefix); err != nil {
			return err
		}
		if _, err := bw.WriteString(r.Value); err != nil {
			return err
		}
		if _, err := bw.WriteRune('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}
