package unpack

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/urlesistiana/v2dat/v2data"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func newGeoIPCmd() *cobra.Command {
	args := new(unpackArgs)
	c := &cobra.Command{
		Use:   "geoip [-o output_dir] [-f tag]... geoip.dat",
		Args:  cobra.ExactArgs(1),
		Short: "Unpack geoip file to text files.",
		Run: func(cmd *cobra.Command, a []string) {
			args.file = a[0]
			if err := unpackGeoIP(args); err != nil {
				logger.Fatal("failed to unpack geoip", zap.Error(err))
			}
		},
		DisableFlagsInUseLine: true,
	}
	c.Flags().StringVarP(&args.outDir, "out", "o", "", "output dir")
	c.Flags().BoolVarP(&args.print, "print", "p", false, "write to stdout instead of files")
	c.Flags().StringArrayVarP(&args.filters, "filter", "f", nil, "unpack given tag")
	return c
}

func unpackGeoIP(args *unpackArgs) error {
	filePath, wantTags, outDir, stdout := args.file, args.filters, args.outDir, args.print
	stdoutMode := outDir == "-" || stdout

	save := func(tag string, geo *v2data.GeoIP) error {
		if stdoutMode {
			fmt.Fprintf(os.Stdout, "# %s (%d cidr)\n", tag, len(geo.GetCidr()))
			return convertV2CidrToText(geo.GetCidr(), os.Stdout)
		}
		file := fmt.Sprintf("%s_%s.txt", fileName(filePath), tag)
		if outDir != "" {
			file = filepath.Join(outDir, file)
		}
		logger.Info("unpacking entry",
			zap.String("tag", tag),
			zap.Int("length", len(geo.GetCidr())),
			zap.String("file", file),
		)
		return convertV2CidrToTextFile(geo.GetCidr(), file)
	}

	if len(wantTags) != 0 {
		return streamGeoIP(filePath, wantTags, save)
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	geoIPList, err := v2data.LoadGeoIPListFromDAT(b)
	if err != nil {
		return err
	}
	for _, geo := range geoIPList.GetEntry() {
		tag := strings.ToLower(geo.GetCountryCode())
		if err := save(tag, geo); err != nil {
			return err
		}
	}
	return nil
}

func streamGeoIP(file string, filters []string, save func(string, *v2data.GeoIP) error) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	want := map[string]struct{}{}
	for _, tag := range filters {
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
		var geo v2data.GeoIP
		if err := proto.Unmarshal(msg, &geo); err != nil {
			return err
		}
		if err := save(tag, &geo); err != nil {
			return err
		}
		got[tag] = struct{}{}
		if len(got) == len(want) {
			return nil
		}
	}
	return nil
}

func convertV2CidrToTextFile(cidr []*v2data.CIDR, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	return convertV2CidrToText(cidr, f)
}

func convertV2CidrToText(cidr []*v2data.CIDR, w io.Writer) error {
	bw := bufio.NewWriter(w)
	for i, record := range cidr {
		ip, ok := netip.AddrFromSlice(record.Ip)
		if !ok {
			return fmt.Errorf("invalid ip at index #%d, %s", i, record.Ip)
		}
		prefix, err := ip.Prefix(int(record.Prefix))
		if !ok {
			return fmt.Errorf("invalid prefix at index #%d, %w", i, err)
		}

		if _, err := bw.WriteString(prefix.String()); err != nil {
			return err
		}
		if _, err := bw.WriteRune('\n'); err != nil {
			return err
		}
	}
	return bw.Flush()
}
