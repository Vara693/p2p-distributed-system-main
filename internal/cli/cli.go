package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"edi_sem2/internal/merkle"
	"edi_sem2/internal/storage"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stderr)
		return errors.New("missing command")
	}

	cmd := args[0]
	switch cmd {
	case "add":
		return runAdd(ctx, args[1:], stdout, stderr)
	case "get":
		return runGet(ctx, args[1:], stdout, stderr)
	case "inspect":
		return runInspect(ctx, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printHelp(stdout)
		return nil
	default:
		printHelp(stderr)
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func defaultStorageDir() string {
	return filepath.Join("storage")
}

func normalizeInterspersedFlags(args []string) []string {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if strings.Contains(a, "=") {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return append(flags, positionals...)
}

func runAdd(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	in := fs.String("in", "", "input file path (or pass as positional arg)")
	storageDir := fs.String("storage", defaultStorageDir(), "storage directory")
	if err := fs.Parse(normalizeInterspersedFlags(args)); err != nil {
		return err
	}
	path := *in
	if path == "" {
		rest := fs.Args()
		if len(rest) != 1 {
			return errors.New("usage: node add <file> [--storage storage]")
		}
		path = rest[0]
	}
	bs, err := storage.NewStore(*storageDir)
	if err != nil {
		return err
	}
	ch := storage.Chunker{ChunkSize: storage.DefaultChunkSize}
	chunks, err := ch.PutFile(bs, path)
	if err != nil {
		return err
	}
	chunkStrs := make([]string, len(chunks))
	for i, c := range chunks {
		chunkStrs[i] = string(c)
	}
	fn := merkle.NewFileNode(chunkStrs)
	nodeBytes, err := merkle.EncodeCanonicalJSON(fn)
	if err != nil {
		return err
	}
	root := storage.NewCID(nodeBytes)
	if _, err := bs.Put(nodeBytes); err != nil {
		return err
	}
	_ = writeFileBundle(bs, root, chunks, nodeBytes)
	_, _ = fmt.Fprintf(stdout, "Root CID: %s\n", root)
	return nil
}

func writeFileBundle(bs *storage.Store, root storage.CID, chunks []storage.CID, nodeBytes []byte) error {
	dir := filepath.Join(bs.RootDir(), "files", string(root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, string(root)+".block"), nodeBytes, 0o644); err != nil {
		return err
	}
	for _, ch := range chunks {
		b, err := bs.Get(ch)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, string(ch)+".block"), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func runGet(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "output file path (default: write to stdout)")
	storageDir := fs.String("storage", defaultStorageDir(), "storage directory")
	if err := fs.Parse(normalizeInterspersedFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("usage: node get <rootCID> [--out file] [--storage storage]")
	}
	root := storage.CID(rest[0])
	bs, err := storage.NewStore(*storageDir)
	if err != nil {
		return err
	}
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := storage.ReconstructFile(ctx, bs, root, f); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(stdout, "Wrote file to %s\n", *out)
		return nil
	}
	return storage.ReconstructFile(ctx, bs, root, stdout)
}

func runInspect(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	storageDir := fs.String("storage", defaultStorageDir(), "storage directory")
	if err := fs.Parse(normalizeInterspersedFlags(args)); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return errors.New("usage: node inspect <cid> [--storage storage]")
	}
	c := storage.CID(rest[0])
	bs, err := storage.NewStore(*storageDir)
	if err != nil {
		return err
	}
	b, err := bs.Get(c)
	if err != nil {
		return err
	}
	var msg string
	if t, err := merkle.DetectType(b); err == nil {
		switch t {
		case merkle.NodeTypeFile:
			var fn merkle.FileNode
			_ = json.Unmarshal(b, &fn)
			msg = fmt.Sprintf("CID: %s\nType: file\nChunks: %d\n", c, len(fn.Chunks))
		case merkle.NodeTypeDirectory:
			var dn merkle.DirectoryNode
			_ = json.Unmarshal(b, &dn)
			msg = fmt.Sprintf("CID: %s\nType: directory\nEntries: %d\n", c, len(dn.Entries))
		default:
			msg = fmt.Sprintf("CID: %s\nType: raw block\nSize: %d bytes\n", c, len(b))
		}
	} else {
		msg = fmt.Sprintf("CID: %s\nType: raw block\nSize: %d bytes\n", c, len(b))
	}
	_, _ = io.WriteString(stdout, msg)
	return nil
}

func printHelp(w io.Writer) {
	_, _ = io.WriteString(w, `node — content-addressed storage (local CLI + networked serve)

Local (no network):
  node add <file> [--storage storage]
  node get <rootCID> [--out file] [--storage storage]
  node inspect <cid> [--storage storage]

Networked node + HTTP API + gRPC:
  node serve --data storage/node1 --grpc 127.0.0.1:50051 --http 127.0.0.1:8080 [--bootstrap http://127.0.0.1:9099] [--replication 3]

`)
}
