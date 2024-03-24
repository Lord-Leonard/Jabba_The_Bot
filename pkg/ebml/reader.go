package ebml

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/bits"
	"time"
)

const (
	ID_EBML                    = 0x1A45DFA3
	ID_Version                 = 0x4286
	ID_ReadVersion             = 0x42F7
	ID_MaxIDLength             = 0x42F2
	ID_MaxSizeLength           = 0x42F3
	ID_DocType                 = 0x4282
	ID_DocTypeVersion          = 0x4287
	ID_DocTypeReadVersion      = 0x4285
	ID_DocTypeExtension        = 0x4281
	ID_DocTypeExtensionName    = 0x4283
	ID_DocTypeExtensionVersion = 0x4284
	ID_CRC32                   = 0xBF
	ID_Void                    = 0xEC
)

type Element struct {
	Id       uint64
	DataSize uint64
	Data     []byte
}

// Header
// Set the EBML characteristics of the data to follow.
// Each EBML Document has to start with this.
type Header struct {
	/*
		Set the EBML characteristics of the data to follow.
		Each EBML Document has to start with this.
	*/
	Version uint8
	/*
		The version of EBML specifications used to create the
		EBML Document.  The version of EBML defined in this document is 1,
		so EBMLVersion SHOULD be 1.
	*/
	ReadVersion uint8
	/*
		The EBMLMaxIDLength Element stores the maximum
		permitted length in octets of the Element IDs to be found within
		the EBML Body.  An EBMLMaxIDLength Element value of four is
		RECOMMENDED, though larger values are allowed
	*/
	MaxIDLength uint8
	/*
		The EBMLMaxSizeLength Element stores the maximum
		permitted length in octets of the expressions of all Element Data
		Sizes to be found within the EBML Body.  The EBMLMaxSizeLength
		Element documents an upper bound for the "length" of all Element
		Data Size expressions within the EBML Body and not an upper bound
		for the "value" of all Element Data Size expressions within the
		EBML Body.  EBML Elements that have an Element Data Size
		expression that is larger in octets than what is expressed by
		EBMLMaxSizeLength Element are invalid.
	*/
	MaxSizeLength uint
	/*
		A string that describes and identifies the content of
		the EBML Body that follows this EBML Header.
	*/
	DocType string
	/*
		The version of DocType interpreter used to create the
		EBML Document.
	*/
	DocTypeVersion uint8
	/*
		The minimum DocType version an EBML Reader has to
		support to read this EBML Document.  The value of the
		DocTypeReadVersion Element MUST be less than or equal to the value
		of the DocTypeVersion Element.
	*/
	DocTypeReadVersion uint8
	/*
		A DocTypeExtension adds extra Elements to the main
		DocType+DocTypeVersion tuple it's attached to.  An EBML Reader MAY
		know these extra Elements and how to use them.  A DocTypeExtension
		MAY be used to iterate between experimental Elements before they
		are integrated into a regular DocTypeVersion.  Reading one
		DocTypeExtension version of a DocType+DocTypeVersion tuple doesn't
		imply one should be able to read upper versions of this
		DocTypeExtension.
	*/
	DocTypeExtension *DocTypeExtension
}

/*
A DocTypeExtension adds extra Elements to the main
DocType+DocTypeVersion tuple it's attached to.  An EBML Reader MAY
know these extra Elements and how to use them.  A DocTypeExtension
MAY be used to iterate between experimental Elements before they
are integrated into a regular DocTypeVersion.  Reading one
DocTypeExtension version of a DocType+DocTypeVersion tuple doesn't
imply one should be able to read upper versions of this
DocTypeExtension.
*/
type DocTypeExtension struct {
	/*
		The name of the DocTypeExtension to differentiate it
		from other DocTypeExtensions of the same DocType+DocTypeVersion
		tuple.  A DocTypeExtensionName value MUST be unique within the
		EBML Header.
	*/
	Name string
	/*
		The version of the DocTypeExtension.  Different
		DocTypeExtensionVersion values of the same DocType +
		DocTypeVersion + DocTypeExtensionName tuple MAY contain completely
		different sets of extra Elements.  An EBML Reader MAY support
		multiple versions of the same tuple, only one version of the
		tuple, or not support the tuple at all.
	*/
	Version uint8
}

type Reader struct {
	r      *bufio.Reader
	src    io.Reader
	seeker io.Seeker
}

func NewReader(r io.Reader) *Reader {
	reader := &Reader{
		r:   bufio.NewReader(r),
		src: r,
	}
	if s, ok := r.(io.Seeker); ok {
		reader.seeker = s
	}
	return reader
}

func (r *Reader) ReadHeader() (*Header, error) {
	root, err := r.readElement()
	if err != nil {
		return nil, err
	}

	if root.Id != ID_EBML {
		return nil, errors.New("expected EBML header")
	}

	headerReader := NewReader(bytes.NewReader(root.Data))
	header := Header{}
	for {
		element, err := headerReader.readElement()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		id := element.Id
		size := element.DataSize

		switch id {
		case ID_Version:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.Version = uint8(val)
			slog.Debug("Version",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", fmt.Sprintf("%d", val),
			)
		case ID_ReadVersion:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.ReadVersion = uint8(val)
			slog.Debug("ReadVersion",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", fmt.Sprintf("%d", val),
			)
		case ID_MaxIDLength:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.MaxIDLength = uint8(val)
			slog.Debug("MaxIDLength",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", fmt.Sprintf("%d", val),
			)
		case ID_MaxSizeLength:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.MaxSizeLength = uint(val)
			slog.Debug("MaxSizeLength",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", val,
			)
		case ID_DocType:
			val, err := element.GetString()
			if err != nil {
				return nil, err
			}
			header.DocType = val
			slog.Debug("DocType",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", val,
			)
		case ID_DocTypeVersion:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.DocTypeVersion = uint8(val)
			slog.Debug("DocTypeVersion",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", val,
			)
		case ID_DocTypeReadVersion:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.DocTypeReadVersion = uint8(val)
			slog.Debug("DocTypeReadVersion",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", val,
			)
		case ID_DocTypeExtension:
			header.DocTypeExtension = &DocTypeExtension{}
			slog.Debug("DocTypeExtension",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
			)
			continue
		case ID_DocTypeExtensionName:
			val, err := element.GetString()
			if err != nil {
				return nil, err
			}
			header.DocTypeExtension.Name = val
			slog.Debug("DocTypeExtensionName",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", val,
			)
		case ID_DocTypeExtensionVersion:
			val, err := element.GetUInt()
			if err != nil {
				return nil, err
			}
			header.DocTypeExtension.Version = uint8(val)
			slog.Debug("DocTypeExtensionVersion",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
				"Value", val,
			)
		default:
			slog.Debug("Unknown/Data Element",
				"ID", fmt.Sprintf("0x%02x", id),
				"Size", fmt.Sprintf("0x%02x", size),
			)

			_, err = r.r.Discard(int(size))
			if err != nil {
				return nil, err
			}
		}
	}

	return &header, nil
}

func (r *Reader) Next() (id uint64, dataSize uint64, err error) {
	for {
		id, dataSize, err = r.readElementHead()
		if err != nil {
			return id, dataSize, err
		}

		if id == ID_Void {
			err := r.Discard(dataSize)
			if err != nil {
				return 0, 0, err
			}
			continue
		}

		if id == ID_CRC32 {
			// TODO: Validate checksum
			err := r.Discard(dataSize)
			if err != nil {
				return 0, 0, err
			}
			continue
		}

		return id, dataSize, err
	}
}

func (e *Element) GetUInt() (uint64, error) {
	if e.Data == nil {
		return 0, fmt.Errorf("element Data is undefined")
	}
	var val uint64
	for _, b := range e.Data {
		val = (val << 8) | uint64(b)
	}
	return val, nil
}

func (e *Element) GetString() (string, error) {
	buf := e.Data
	if len(buf) > 0 && buf[len(buf)-1] == 0 {
		buf = buf[:len(buf)-1]
	}
	return string(buf), nil
}

func (r *Reader) ReadVInt(v bool) (value uint64, bytesRead int, err error) {
	firstByte, err := r.r.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	if firstByte == 0 {
		return 0, 0, fmt.Errorf("invalid vint")
	}
	length := bits.LeadingZeros8(firstByte) + 1
	maskValue := byte(0xFF >> length)

	value = uint64(firstByte)
	if v {
		value = uint64(firstByte & maskValue)
	}

	for i := 1; i < length; i++ {
		b, err := r.r.ReadByte()
		if err != nil {
			return 0, 0, err
		}
		value = (value << 8) | uint64(b)
	}

	return value, length, nil
}

func (r *Reader) readElementHead() (id uint64, size uint64, err error) {
	id, _, err = r.ReadVInt(false)
	if err != nil {
		return 0, 0, err
	}

	size, _, err = r.ReadVInt(true)
	if err == io.EOF {
		return 0, 0, fmt.Errorf("unexpected end of file")
	}
	if err != nil {
		return 0, 0, err
	}

	return id, size, nil
}

func (r *Reader) readElement() (*Element, error) {
	element := &Element{}

	id, _, err := r.ReadVInt(false)
	if err != nil {
		return nil, err
	}
	element.Id = id

	size, _, err := r.ReadVInt(true)
	if err == io.EOF {
		return nil, fmt.Errorf("unexpected end of file")
	}
	if err != nil {
		return nil, err
	}
	element.DataSize = size

	data := make([]byte, size)
	if _, err := io.ReadFull(r.r, data); err != nil {
		return nil, err
	}
	element.Data = data

	return element, nil
}

func (r *Reader) ReadBinary(size uint64) ([]byte, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(r.r, data); err != nil {
		return nil, err
	}
	return data, nil
}

func (r *Reader) ReadUInt(size uint64) (uint64, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(r.r, data); err != nil {
		return 0, err
	}

	var val uint64
	for _, b := range data {
		val = (val << 8) | uint64(b)
	}
	return val, nil
}
func (r *Reader) ReadInt(size uint64) (int64, error) {
	if size == 0 || size > 8 {
		return 0, fmt.Errorf("invalid int size %d", size)
	}

	var val int64
	for i := uint64(0); i < size; i++ {
		b, err := r.r.ReadByte()
		if err != nil {
			return 0, err
		}
		val = (val << 8) | int64(b)
	}

	shift := 64 - size*8
	val = (val << shift) >> shift

	return val, nil
}

func (r *Reader) ReadFloat(size uint64) (float64, error) {
	switch size {
	case 4:
		var buf [4]byte
		if _, err := io.ReadFull(r.r, buf[:]); err != nil {
			return 0, err
		}
		bitsBuffer := binary.BigEndian.Uint32(buf[:])
		return float64(math.Float32frombits(bitsBuffer)), nil

	case 8:
		var buf [8]byte
		if _, err := io.ReadFull(r.r, buf[:]); err != nil {
			return 0, err
		}
		bitsBuffer := binary.BigEndian.Uint64(buf[:])
		return math.Float64frombits(bitsBuffer), nil

	default:
		return 0, fmt.Errorf("invalid float size %d", size)
	}
}

func (r *Reader) Discard(size uint64) error {
	_, err := r.r.Discard(int(size))
	if err != nil {
		return err
	}
	return nil
}

func (r *Reader) ReadString(size uint64) (string, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(r.r, data); err != nil {
		return "", err
	}

	if len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	return string(data), nil
}

var matroskaEpoch = time.Date(
	2001, 1, 1, 0, 0, 0, 0, time.UTC,
)

func (r *Reader) ReadDate(size uint64) (time.Time, error) {
	if size != 8 {
		return time.Time{}, fmt.Errorf("invalid DateUTC size %d", size)
	}

	var buf [8]byte
	if _, err := io.ReadFull(r.r, buf[:]); err != nil {
		return time.Time{}, err
	}

	ns := int64(binary.BigEndian.Uint64(buf[:]))
	return matroskaEpoch.Add(time.Duration(ns)), nil
}

func (r *Reader) Tell() (int64, error) {
	if r.seeker == nil {
		return 0, fmt.Errorf("reader is not seekable")
	}
	absolute, err := r.seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	return absolute - int64(r.r.Buffered()), nil
}

func (r *Reader) CanSeek() bool {
	return r.seeker != nil
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	if r.seeker == nil {
		return 0, fmt.Errorf("reader is not seekable")
	}

	newPos, err := r.seeker.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	r.r = bufio.NewReader(r.src)
	return newPos, nil
}
