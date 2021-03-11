package types

import (
	"fmt"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"
	"io"
)

// Ideally this would be a tag on the struct field telling cbor-gen to enforce higher max-len
type ManyBytes struct {
	b []byte
}

func NewManyBytes(b []byte) *ManyBytes {
	return &ManyBytes{b: b}
}
func (t *ManyBytes) Bytes() []byte {
	if t == nil {
		return nil
	}
	return t.b
}

const many = 100 << 20

func (t *ManyBytes) MarshalCBOR(w io.Writer) error {
	if t == nil {
		t = &ManyBytes{}
	}

	if len(t.b) > many {
		return xerrors.Errorf("byte array in field t.Result was too long")
	}

	scratch := make([]byte, 9)

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajByteString, uint64(len(t.b))); err != nil {
		return err
	}

	if _, err := w.Write(t.b[:]); err != nil {
		return err
	}
	return nil
}

func (t *ManyBytes) UnmarshalCBOR(r io.Reader) error {
	*t = ManyBytes{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 9)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}

	if extra > many {
		return fmt.Errorf("byte array too large (%d)", extra)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("expected byte array")
	}

	if extra > 0 {
		t.b = make([]uint8, extra)
	}

	if _, err := io.ReadFull(br, t.b[:]); err != nil {
		return err
	}

	return nil
}
