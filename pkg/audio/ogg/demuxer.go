package ogg

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

type Demuxer struct {
	reader       *bufio.Reader
	packetQueue  [][]byte
	packetBuffer []byte
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		reader:       bufio.NewReader(r),
		packetQueue:  make([][]byte, 0),
		packetBuffer: make([]byte, 0),
	}
}

func (demuxer *Demuxer) ProvideFrame() ([]byte, error) {
	if len(demuxer.packetQueue) > 0 {
		pkt := demuxer.packetQueue[0]
		demuxer.packetQueue = demuxer.packetQueue[1:]
		return pkt, nil
	}

	for {
		page, err := demuxer.readPage()
		if err != nil {
			return nil, err
		}

		dataIdx := 0
		for _, segmentLen := range page.Header.SegmentTable {
			val := int(segmentLen)
			segment := page.Data[dataIdx : dataIdx+val]
			demuxer.packetBuffer = append(demuxer.packetBuffer, segment...)
			dataIdx += val

			// wenn segmentLength < 255
			if val < 255 {
				finalPacket := make([]byte, len(demuxer.packetBuffer))
				copy(finalPacket, demuxer.packetBuffer)
				demuxer.packetQueue = append(demuxer.packetQueue, finalPacket)

				demuxer.packetBuffer = demuxer.packetBuffer[:0]
			}
		}

		if len(demuxer.packetQueue) > 0 {
			packet := demuxer.packetQueue[0]
			demuxer.packetQueue = demuxer.packetQueue[1:]
			return packet, nil
		}
	}

}

type Page struct {
	Header Header
	Data   []byte
}

/*
0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1| Byte
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| capture_pattern: Magic number for page start "OggS"           | 0-3
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| version       | header_type   | granule_position              | 4-7
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               | 8-11
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                               | bitstream_serial_number       | 12-15
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                               | page_sequence_number          | 16-19
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                               | CRC_checksum                  | 20-23
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                               |page_segments  | segment_table | 24-27
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| ...                                                           | 28-
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
*/

type PageFlag uint8

const (
	FlagContinuation PageFlag = 0x01
	FlagBOS          PageFlag = 0x02 // Beginning of Stream
	FlagEOS          PageFlag = 0x04 // End of Stream
)

type Header struct {
	/*
		1 Byte signifying the version number of
		gg file format used in this stream
	*/
	Version uint8
	/*
		the bits in this 1-Byte field identify the specific type of this page.
			*  bit 0x01

			 set: page contains data of a packet continued from the previous page

			 unset: page contains a fresh packet
			*  bit 0x02

			 set: this is the first page of a logical bitstream (bos)

			 unset: this page is not a first page
			*  bit 0x04

			 set: this is the last page of a logical bitstream (eos)

			 unset: this page is not a last page
	*/
	Type uint8
	/*
		8-Byte field containing position information.
		For example, for an audio stream, it MAY contain the total number
		of PCM samples encoded after including all frames finished on this
		page.  For a video stream it MAY contain the total number of video
		frames encoded after this page.  This is a hint for the decoder
		and gives it some timing and position information.  Its meaning is
		dependent on the codec for that logical bitstream and specified in
		a specific media mapping.  A special value of -1 (in two's
		complement) indicates that no packets finish on this page.
	*/
	GranulePosition uint64
	/*
		4 Byte field containing the unique
		serial number by which the logical bitstream is identified
	*/
	BitstreamSerialNumber uint32
	/*
		4 Byte field containing the sequence
		number of the page so the decoder can identify page loss.  This
		sequence number is increasing on each logical bitstream
		separately.
	*/
	PageSequenceNumber uint32
	/*
		4 Byte field containing a 32-bit CRC checksum of
		the page (including header with zero CRC field and page content).
		The generator polynomial is 0x04c11db7.
	*/
	CRCChecksum uint32
	/*
		1 Byte giving the number of segment entries
		encoded in the segment table.
	*/
	NumberPageSegments uint8
	/*
		Number_page_segments Bytes containing the lacing
		values of all segments in this page.  Each Byte contains one
		lacing value.
	*/
	SegmentTable []byte
}

func (demuxer *Demuxer) readPage() (Page, error) {
	staticHeader := make([]byte, 27)
	_, err := io.ReadFull(demuxer.reader, staticHeader)
	if err != nil {
		return Page{}, err
	}

	if string(staticHeader[:4]) != "OggS" {
		return Page{}, fmt.Errorf("invalid ogg staticHeader, got %v", staticHeader[:4])
	}

	page := Page{
		Header: Header{
			Version:               staticHeader[4],
			Type:                  staticHeader[5],
			GranulePosition:       binary.LittleEndian.Uint64(staticHeader[6:14]),
			BitstreamSerialNumber: binary.LittleEndian.Uint32(staticHeader[14:18]),
			PageSequenceNumber:    binary.LittleEndian.Uint32(staticHeader[18:22]),
			CRCChecksum:           binary.LittleEndian.Uint32(staticHeader[22:26]),
			NumberPageSegments:    staticHeader[26],
		}}

	segmentsTable := make([]byte, page.Header.NumberPageSegments)
	_, err = io.ReadFull(demuxer.reader, segmentsTable)
	if err != nil {
		return page, err
	}
	page.Header.SegmentTable = segmentsTable

	bodySize := 0
	for _, segment := range segmentsTable {
		bodySize += int(segment)
	}

	data := make([]byte, bodySize)
	_, err = io.ReadFull(demuxer.reader, data)
	if err != nil {
		return page, err
	}
	page.Data = data

	return page, nil
}
