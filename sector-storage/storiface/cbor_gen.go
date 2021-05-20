// Code generated by github.com/whyrusleeping/cbor-gen. DO NOT EDIT.

package storiface

import (
	"fmt"
	"io"
	"sort"

	cid "github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	xerrors "golang.org/x/xerrors"
)

var _ = xerrors.Errorf
var _ = cid.Undef
var _ = sort.Sort

func (t *CallID) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if _, err := w.Write([]byte{162}); err != nil {
		return err
	}

	scratch := make([]byte, 9)

	// t.Sector (abi.SectorID) (struct)
	if len("Sector") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"Sector\" was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len("Sector"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("Sector")); err != nil {
		return err
	}

	if err := t.Sector.MarshalCBOR(w); err != nil {
		return err
	}

	// t.ID (uuid.UUID) (array)
	if len("ID") > cbg.MaxLength {
		return xerrors.Errorf("Value in field \"ID\" was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajTextString, uint64(len("ID"))); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string("ID")); err != nil {
		return err
	}

	if len(t.ID) > cbg.ByteArrayMaxLen {
		return xerrors.Errorf("Byte array in field t.ID was too long")
	}

	if err := cbg.WriteMajorTypeHeaderBuf(scratch, w, cbg.MajByteString, uint64(len(t.ID))); err != nil {
		return err
	}

	if _, err := w.Write(t.ID[:]); err != nil {
		return err
	}
	return nil
}

func (t *CallID) UnmarshalCBOR(r io.Reader) error {
	*t = CallID{}

	br := cbg.GetPeeker(r)
	scratch := make([]byte, 8)

	maj, extra, err := cbg.CborReadHeaderBuf(br, scratch)
	if err != nil {
		return err
	}
	if maj != cbg.MajMap {
		return fmt.Errorf("cbor input should be of type map")
	}

	if extra > cbg.MaxLength {
		return fmt.Errorf("CallID: map struct too large (%d)", extra)
	}

	var name string
	n := extra

	for i := uint64(0); i < n; i++ {

		{
			sval, err := cbg.ReadStringBuf(br, scratch)
			if err != nil {
				return err
			}

			name = string(sval)
		}

		switch name {
		// t.Sector (abi.SectorID) (struct)
		case "Sector":

			{

				if err := t.Sector.UnmarshalCBOR(br); err != nil {
					return xerrors.Errorf("unmarshaling t.Sector: %w", err)
				}

			}
			// t.ID (uuid.UUID) (array)
		case "ID":

			maj, extra, err = cbg.CborReadHeaderBuf(br, scratch)
			if err != nil {
				return err
			}

			if extra > cbg.ByteArrayMaxLen {
				return fmt.Errorf("t.ID: byte array too large (%d)", extra)
			}
			if maj != cbg.MajByteString {
				return fmt.Errorf("expected byte array")
			}

			if extra != 16 {
				return fmt.Errorf("expected array to have 16 elements")
			}

			t.ID = [16]uint8{}

			if _, err := io.ReadFull(br, t.ID[:]); err != nil {
				return err
			}

		default:
			// Field doesn't exist on this type, so ignore it
			cbg.ScanForLinks(r, func(cid.Cid) {})
		}
	}

	return nil
}
