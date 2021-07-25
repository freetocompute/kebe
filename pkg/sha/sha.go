package sha

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"github.com/snapcore/snapd/osutil"
	"io"
)

const (
	hashDigestBufSize = 2 * 1024 * 1024
)

// SnapFileSHA3_384 computes the SHA3-384 digest of the given snap file.
// It also returns its size.
func SnapFileSHA3_384(snapPath string) (digest string, size uint64, err error) {
	sha3_384Dgst, size, err := osutil.FileDigest(snapPath, crypto.SHA3_384)
	if err != nil {
		return "", 0, fmt.Errorf("cannot compute snap %q digest: %v", snapPath, err)
	}

	sha3_384, err := EncodeDigest(crypto.SHA3_384, sha3_384Dgst)
	if err != nil {
		return "", 0, fmt.Errorf("cannot encode snap %q digest: %v", snapPath, err)
	}
	return sha3_384, size, nil
}

// EncodeDigest encodes the digest from hash algorithm to be put in an assertion header.
func EncodeDigest(hash crypto.Hash, hashDigest []byte) (string, error) {
	algo := ""
	switch hash {
	case crypto.SHA512:
		algo = "sha512"
	case crypto.SHA3_384:
		algo = "sha3-384"
	default:
		return "", fmt.Errorf("unsupported hash")
	}
	if len(hashDigest) != hash.Size() {
		return "", fmt.Errorf("hash digest by %s should be %d bytes", algo, hash.Size())
	}
	return base64.RawURLEncoding.EncodeToString(hashDigest), nil
}

// FileDigest computes a hash digest of the file using the given hash.
// It also returns the file size.
func FileDigest(reader io.Reader, hash crypto.Hash) ([]byte, uint64, error) {
	h := hash.New()
	size, err := io.CopyBuffer(h, reader, make([]byte, hashDigestBufSize))
	if err != nil {
		return nil, 0, err
	}
	return h.Sum(nil), uint64(size), nil
}

// SnapFileSHA3_384FromReader computes the SHA3-384 digest of the given snap file.
// It also returns its size.
func SnapFileSHA3_384FromReader(reader io.Reader) (digest string, size uint64, err error) {
	sha3_384Dgst, size, err := FileDigest(reader, crypto.SHA3_384)

	sha3_384, err := EncodeDigest(crypto.SHA3_384, sha3_384Dgst)
	if err != nil {
		return "", 0, fmt.Errorf("%s", "cannot encode snap")
	}
	return sha3_384, size, nil
}
