package fireproxy

import (
	"bufio"
	"bytes"
	"io"
)

type modelRewriteReadCloser struct {
	body         io.ReadCloser
	reader       *bufio.Reader
	replacements map[string][]byte
	pending      []byte
	drained      bool
}

func newModelRewriteReadCloser(body io.ReadCloser, cfg Config) io.ReadCloser {
	replacements := make(map[string][]byte, len(cfg.UpstreamModelAliases))
	for alias := range cfg.UpstreamModelAliases {
		replacements[alias] = []byte(cfg.PublicModelID)
	}
	return &modelRewriteReadCloser{
		body:         body,
		reader:       bufio.NewReader(body),
		replacements: replacements,
	}
}

func (r *modelRewriteReadCloser) Read(p []byte) (int, error) {
	for len(r.pending) == 0 && !r.drained {
		line, err := r.reader.ReadBytes('\n')
		if len(line) > 0 {
			rewritten := line
			for alias, replacement := range r.replacements {
				rewritten = bytes.ReplaceAll(rewritten, []byte(alias), replacement)
			}
			r.pending = append(r.pending, rewritten...)
		}
		if err == io.EOF {
			r.drained = true
			break
		}
		if err != nil {
			return 0, err
		}
	}

	if len(r.pending) == 0 && r.drained {
		return 0, io.EOF
	}

	n := copy(p, r.pending)
	r.pending = r.pending[n:]
	return n, nil
}

func (r *modelRewriteReadCloser) Close() error {
	return r.body.Close()
}
