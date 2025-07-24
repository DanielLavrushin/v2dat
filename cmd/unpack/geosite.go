package unpack

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/urlesistiana/v2dat/v2data"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
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
		Use:   "geosite [-o output_dir] [-p] [-f tag[@attr]...]... geosite.dat",
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
	filePath, suffixes, outDir, stdout := args.file, args.filters, args.outDir, args.print
	stdoutMode := outDir == "-" || stdout

	save := func(suffix string, domains []*v2data.Domain) error {
		if stdoutMode {
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
	if len(suffixes) != 0 {
		return streamGeoSite(filePath, suffixes, save)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	geoSiteList, err := v2data.LoadGeoSiteList(data)
	if err != nil {
		return err
	}
	entries := make(map[string][]*v2data.Domain, len(geoSiteList.GetEntry()))
	for _, gs := range geoSiteList.GetEntry() {
		tag := strings.ToLower(gs.GetCountryCode())
		entries[tag] = gs.GetDomain()
	}
	for tag, domains := range entries {
		if err := save(tag, domains); err != nil {
			return fmt.Errorf("failed to save %s: %w", tag, err)
		}
	}
	return nil
}

func readCountryCode(msg []byte) (string, error) {
	if len(msg) == 0 || msg[0] != 0x0A {
		return "", fmt.Errorf("bad key")
	}
	l, n := binary.Uvarint(msg[1:])
	if n <= 0 {
		return "", fmt.Errorf("bad varint")
	}
	start := 1 + n
	end := start + int(l)
	if end > len(msg) {
		return "", fmt.Errorf("string truncated")
	}
	return strings.ToLower(string(msg[start:end])), nil
}

func streamGeoSite(file string, filters []string, save func(string, []*v2data.Domain) error) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	want := map[string]struct{}{}
	for _, s := range filters {
		tag, _ := splitAttrs(s)
		want[strings.ToLower(tag)] = struct{}{}
	}
	got := map[string]struct{}{}
	r := bufio.NewReaderSize(f, 32*1024)
	for {
		tagByte, err := r.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if tagByte != 0x0A {
			return fmt.Errorf("unexpected wire tag %02X", tagByte)
		}
		length, err := binary.ReadUvarint(r)
		if err != nil {
			return err
		}
		msg := make([]byte, length)
		if _, err := io.ReadFull(r, msg); err != nil {
			return err
		}
		tag, err := readCountryCode(msg)
		if err != nil {
			return err
		}
		if _, ok := want[tag]; !ok {
			continue
		}
		var gs v2data.GeoSite
		if err := proto.Unmarshal(msg, &gs); err != nil {
			return err
		}
		if err := save(tag, gs.GetDomain()); err != nil {
			return err
		}
		got[tag] = struct{}{}
		if len(got) == len(want) {
			return nil
		}
	}
	return nil
}

func convertV2DomainToText(dom []*v2data.Domain, w io.Writer) error {
	b := strings.Builder{}
	// crude pre‑size: avg 30 bytes per line
	b.Grow(len(dom) * 30)

	for _, d := range dom {
		switch d.Type {
		case v2data.Domain_Plain:
			b.WriteString("keyword:")
		case v2data.Domain_Regex:
			b.WriteString("regexp:")
		case v2data.Domain_Full:
			b.WriteString("full:")
		}
		b.WriteString(d.Value)
		b.WriteByte('\n')
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func convertV2DomainToTextFile(domain []*v2data.Domain, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	return convertV2DomainToText(domain, f)
}
