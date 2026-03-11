package webm

import (
	"Jabba_The_Bot/pkg/ebml"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"
	"time"
)

type DemuxerOption func(*Demuxer)

type Demuxer struct {
	r *ebml.Reader
	c io.Closer

	EbmlHeader *ebml.Header
	State      Segment
	debugLog   *slog.Logger

	mu                   sync.RWMutex
	opMu                 sync.Mutex
	timestampScale       uint64
	currentTimestamp     int64
	seekPoints           []seekPoint
	currentClusterOffset int64
	segmentDataStart     int64
	segmentStartSet      bool
	pendingFrame         []byte
	duration             time.Duration
	durationSet          bool
	closeOnce            sync.Once
}

type seekPoint struct {
	Timestamp int64
	Offset    int64
}

/*
BlockMore

Contains the BlockAdditional and some parameters.
*/
type BlockMore struct {
	/*
		Interpreted by the codec as it wishes (using the
		BlockAddID).
	*/
	BlockAdditional []byte
	/*
		An ID that identifies how to interpret the
		BlockAdditional data; see Section 4.1.5 of [MatroskaCodec] for
		more information.  A value of 1 indicates that the BlockAdditional
		data is defined by the codec.  Any other value indicates that the
		BlockAdditional data should be handled according to the
		BlockAddIDType that is located in the TrackEntry.
	*/
	BlockAddID uint64
}

/*
BlockAdditions

Contains additional binary data to complete the Block
element; see Section 4.1.5 of [MatroskaCodec] for more
information.  An EBML parser that has no knowledge of the Block
structure could still see and use/skip these data.
*/
type BlockAdditions struct {
	BlockMore BlockMore
}

/*
	BlockGroup

Basic container of information containing a single Block
and information specific to that Block.
*/
type BlockGroup struct {
	/*
		Block containing the actual data to be rendered and a
		timestamp relative to the Cluster Timestamp; see Section 10.1 on
		Block Structure.
	*/
	Block []byte

	BlockAdditions BlockAdditions
	/*
		The duration of the Block, expressed in Track Ticks; see
		Section 11.1.  The BlockDuration element can be useful at the end
		of a Track to define the duration of the last frame (as there is
		no subsequent Block available) or when there is a break in a track
		like for subtitle tracks.

		+===========+==================================================+
		| attribute | note                                             |
		+===========+==================================================+
		| minOccurs | BlockDuration MUST be set (minOccurs=1) if the   |
		|           | associated TrackEntry stores a DefaultDuration   |
		|           | value.                                           |
		+-----------+--------------------------------------------------+
		| default   | If a value is not present and no DefaultDuration |
		|           | is defined, the value is assumed to be the       |
		|           | difference between the timestamp of this Block   |
		|           | and the timestamp of the next Block in "display" |
		|           | order (not coding order).                        |
		+-----------+--------------------------------------------------+
	*/
	BlockDuration uint64
	/*
		This frame is referenced and has the specified cache
		priority.  In the cache, only a frame of the same or higher
		priority can replace this frame.  A value of 0 means the frame is
		not referenced.
	*/
	ReferencePriority uint64
	/*
		A timestamp value, relative to the timestamp of the
		Block in this BlockGroup, expressed in Track Ticks; see
		Section 11.1. This is used to reference other frames necessary to
		decode this frame.  The relative value SHOULD correspond to a
		valid Block that this Block depends on.  Historically, Matroska
		Writers didn't write the actual Block(s) that this Block depends
		on, but they did write _some_ Block(s) in the past.

		The value "0" MAY also be used to signify that this Block cannot be
		decoded on its own, but the necessary reference Block(s) is unknown.
		In this case, other ReferenceBlock elements MUST NOT be found in the
		same BlockGroup.  If the BlockGroup doesn't have a ReferenceBlock
		element, then the Block it contains can be decoded without using any
		other Block data.
	*/
	ReferenceBlock int64
	/*
		The new codec state to use.  Data interpretation is
		private to the codec.  This information SHOULD always be
		referenced by a seek entry.
	*/
	CodecState []byte
	/*
		Duration of the silent data added to the Block,
		expressed in Matroska Ticks -- i.e., in nanoseconds; see
		Section 11.1 (padding at the end of the Block for positive values
		and at the beginning of the Block for negative values).  The
		duration of DiscardPadding is not calculated in the duration of
		the TrackEntry and SHOULD be discarded during playback.
	*/
	DiscardPadding int64
}

/*
	Cluster

The Top-Level Element containing the (monolithic) Block
structure.
*/
type Cluster struct {
	/*
		Absolute timestamp of the cluster, expressed in Segment
		Ticks, which are based on TimestampScale
	*/
	Timestamp uint64
	/*
		The Segment Position of the Cluster in the Segment (0 in
		live streams).  It might help to resynchronize the offset on
		damaged streams.
	*/
	Position uint64
	/*
		Size of the previous Cluster, in octets.  Can be useful
		for backward playing.
	*/
	PrevSize uint64
	/*
		Similar to Block (see Section 10.1) but without all the
		extra information.  Mostly used to reduce overhead when no extra
		feature is needed; see Section 10.2 on SimpleBlock Structure.
	*/
	SimpleBlock []byte

	BlockGroup BlockGroup
}

/*
TrackEntry

Describes a track with all elements.
*/
type TrackEntry struct {
	/*
		The track number as used in the Block Header.
	*/
	Number uint64
	/*
		A UID that identifies the Track.
	*/
	UID uint64
}

/*
Tracks

A Top-Level Element of information with many tracks
described.
*/
type Tracks struct {
	TrackEntries []*TrackEntry
}

/*
CueReference

The Clusters containing the referenced Blocks.
*/
type CueReference struct {
	/*
		Timestamp of the referenced Block, expressed in Segment
		Ticks which is based on TimestampScale; see Section 11.1.
	*/
	CueRefTime uint64
}

/*
CueTrackPosition

Contains positions for different tracks corresponding to
the timestamp.
*/
type CueTrackPosition struct {
	/*
		The track for which a position is given.
	*/
	CueTrack uint64
	/*
		The Segment Position (Section 16) of the Cluster
		containing the associated Block.
	*/
	CueClusterPosition uint64
	/*
		The relative position inside the Cluster of the
		referenced SimpleBlock or BlockGroup with 0 being the first
		possible position for an element inside that Cluster.
	*/
	CueRelativePosition uint64
	/*
		The duration of the block, expressed in Segment Ticks,
		which are based on TimestampScale; see Section 11.1.  If missing,
		the track's DefaultDuration does not apply and no duration
		information is available in terms of the cues.
	*/
	CueDuration uint64
	/*
		Number of the Block in the specified Cluster.
	*/
	CueBlockNumber uint64
	/*
		The Segment Position (Section 16) of the Codec State
		corresponding to this Cues element. 0 means that the data is taken
		from the initial TrackEntry.
	*/
	CueCodecState uint64

	CueReference CueReference
}

/*
CuesPoint
Contains all information relative to a seek point in the
Segment.
*/
type CuesPoint struct {
	/*
		Absolute timestamp of the seek point, expressed in
		Segment Ticks, which are based on TimestampScale; see
		Section 11.1.
	*/
	CueTime           uint64
	CueTrackPositions []*CueTrackPosition
}

/*
Cues

A Top-Level Element to speed seeking access.  All
entries are local to the Segment.
*/
type Cues struct {
	CuesPoint CuesPoint
}

/*
	Segment

The Root Element that contains all other Top-Level
Elements;
*/
type Segment struct {
	SeekHead []*SeekHead
	Info     Info
	Tracks   Tracks
	Cluster  Cluster
	Cues     Cues
}

/*
	Info

Contains general information about the Segment.
*/
type Info struct {
	/*
		A randomly generated UID that identifies the Segment
		amongst many others (128 bits).  It is equivalent to a Universally
		Unique Identifier (UUID) v4 [RFC9562] with all bits randomly (or
		pseudorandomly) chosen.  An actual UUID v4 value, where some bits
		are not random, MAY also be used.

		usage notes: If the Segment is a part of a Linked Segment, then this
		element is REQUIRED.  The value of the UID MUST contain at least
		one bit set to 1.
	*/
	SegmentUUID []byte
	/* A filename corresponding to this Segment. */
	SegmentFilename string
	/*
		An ID that identifies the previous Segment of a Linked
		Segment.

		usage notes: If the Segment is a part of a Linked Segment that uses
		Hard Linking (Section 17.1), then either the PrevUUID or the
		NextUUID element is REQUIRED.  If a Segment contains a PrevUUID
		but not a NextUUID, then it MAY be considered as the last Segment
		of the Linked Segment.  The PrevUUID MUST NOT be equal to the
		SegmentUUID.
	*/
	PrevUUID []byte
	/*
		A filename corresponding to the file of the previous
		Linked Segment.

		usage notes: Provision of the previous filename is for display
		convenience, but PrevUUID SHOULD be considered authoritative for
		identifying the previous Segment in a Linked Segment.
	*/
	PrevFilename string
	/*
		An ID that identifies the next Segment of a Linked
		Segment.

		usage notes: If the Segment is a part of a Linked Segment that uses
		Hard Linking (Section 17.1), then either the PrevUUID or the
		NextUUID element is REQUIRED.  If a Segment contains a NextUUID
		but not a PrevUUID, then it MAY be considered as the first Segment
		of the Linked Segment.  The NextUUID MUST NOT be equal to the
		SegmentUUID.
	*/
	NextUUID []byte
	/*
		A filename corresponding to the file of the next Linked
		Segment.

		usage notes: Provision of the next filename is for display
		convenience, but NextUUID SHOULD be considered authoritative for
		identifying the Next Segment.
	*/
	NextFilename string
	/*
		A UID that all Segments of a Linked Segment MUST share
		(128 bits).  It is equivalent to a UUID v4 [RFC9562] with all bits
		randomly (or pseudorandomly) chosen.  An actual UUID v4 value,
		where some bits are not random, MAY also be used.

		usage notes: If the Segment Info contains a ChapterTranslate
		element, this element is REQUIRED.
	*/
	SegmentFamily []byte

	ChapterTranslate ChapterTranslate
	/*
		Base unit for Segment Ticks and Track Ticks, in
		nanoseconds.  A TimestampScale value of 1000000 means scaled
		timestamps in the Segment are expressed in milliseconds; see
		Section 11 on how to interpret timestamps.
	*/
	TimestampScale uint64
	/*
		Duration of the Segment, expressed in Segment Ticks,
		which are based on TimestampScale
	*/
	Duration float64
	/*
		The date and time that the Segment was created by the
		muxing application or library.
	*/
	DateUTC time.Time
	/*
		General name of the Segment.
	*/
	Title string
	/*
		Muxing application or library (example: "libmatroska-
		0.4.3").
	*/
	MuxingApp string
	/*
		Writing application (example: "mkvmerge-0.3.3").
	*/
	WritingApp string
}

/*
	ChapterTranslate

The mapping between this Segment and a segment value in
the given Chapter Codec.
rationale:  Chapter Codecs may need to address different segments,
but they may not know of the way to identify such segments when
stored in Matroska.  This element and its child elements add a way
to map the internal segments known to the Chapter Codec to the
SegmentUUIDs in Matroska.  This allows remuxing a file with
Chapter Codec without changing the content of the codec data, just
the Segment mapping.
*/
type ChapterTranslate struct {
	/*
		The binary value used to represent this Segment in the
		chapter codec data.  The format depends on the ChapProcessCodecID
		used
	*/
	ID []byte
	/*
		Applies to the chapter codec of the given chapter
		edition(s)
	*/
	Codec uint64
	/*
		When no ChapterTranslateEditionUID is specified in the
		ChapterTranslate, the ChapterTranslate applies to all chapter
		editions found in the Segment using the given
		ChapterTranslateCodec.
	*/
	EditionUID uint64
}

/*
	SeekHead

Contains seeking information of Top-Level Elements.
*/
type SeekHead struct {
	Seek []*Seek
}

/*
	Seek

Contains a single seek entry to an EBML Element.
*/
type Seek struct {
	/* The binary EBML ID of a Top-Level Element. */
	ID []byte
	/* he Segment Position (Section 16) of a Top-Level Element. */
	Position uint64
}

func WithDebugLogger(logger *slog.Logger) DemuxerOption {
	return func(d *Demuxer) {
		d.debugLog = logger
	}
}

func NewDemuxer(r io.Reader, options ...DemuxerOption) (*Demuxer, error) {
	var closer io.Closer
	if c, ok := r.(io.Closer); ok {
		closer = c
	}

	demuxer := &Demuxer{
		c:                    closer,
		timestampScale:       1000000,
		seekPoints:           make([]seekPoint, 0, 32),
		currentClusterOffset: -1,
		segmentDataStart:     -1,
	}

	for _, option := range options {
		if option == nil {
			continue
		}
		option(demuxer)
	}

	ebmlReader := ebml.NewReader(r, ebml.WithDebugLogger(demuxer.debugLog))

	header, err := ebmlReader.ReadHeader()
	if err != nil {
		return nil, err
	}
	if header.DocType != "webm" {
		return nil, fmt.Errorf("unknown DocType: %s", header.DocType)
	}

	demuxer.r = ebmlReader
	demuxer.EbmlHeader = header

	return demuxer, nil

}

func (d *Demuxer) logDebug(msg string, args ...any) {
	d.log(slog.LevelDebug, msg, args...)
}

func (d *Demuxer) log(level slog.Level, msg string, args ...any) {
	if d == nil || d.debugLog == nil {
		return
	}
	d.debugLog.Log(context.Background(), level, msg, args...)
}

// TODO: should it have that? Is that a demuxer concern?
func (d *Demuxer) Close() error {
	var closeErr error
	d.closeOnce.Do(func() {
		if d.c != nil {
			closeErr = d.c.Close()
		}
	})
	return closeErr
}

func (d *Demuxer) Duration() time.Duration {
	d.mu.RLock()
	if d.durationSet {
		defer d.mu.RUnlock()
		return d.duration
	}
	scale := d.timestampScale
	infoDuration := d.State.Info.Duration
	d.mu.RUnlock()

	if infoDuration <= 0 {
		return 0
	}
	if scale == 0 {
		scale = 1000000
	}
	durationNs := infoDuration * float64(scale)
	d.mu.Lock()
	d.duration = time.Duration(durationNs)
	d.durationSet = true
	d.mu.Unlock()
	return time.Duration(durationNs)
}

func (d *Demuxer) Position() time.Duration {
	d.mu.RLock()
	scale := d.timestampScale
	timestamp := d.currentTimestamp
	d.mu.RUnlock()
	if scale == 0 {
		scale = 1000000
	}
	if timestamp < 0 {
		timestamp = 0
	}
	return time.Duration(timestamp) * time.Duration(scale)
}

func (d *Demuxer) Seek(target time.Duration) error {
	d.opMu.Lock()
	defer d.opMu.Unlock()

	if !d.r.CanSeek() {
		return fmt.Errorf("demuxer source is not seekable")
	}
	if target < 0 {
		target = 0
	}

	d.mu.RLock()
	scale := d.timestampScale
	if scale == 0 {
		scale = 1000000
	}
	targetTicks := int64(target / time.Duration(scale))
	point, ok := d.findSeekPointLocked(targetTicks)
	d.mu.RUnlock()
	if !ok {
		return fmt.Errorf("seek index not available yet")
	}

	if _, err := d.r.Seek(point.Offset, io.SeekStart); err != nil {
		return err
	}

	d.mu.Lock()
	d.State.Cluster = Cluster{}
	d.currentTimestamp = point.Timestamp
	d.pendingFrame = nil
	d.mu.Unlock()

	for {
		packet, err := d.provideFrameLocked()
		if err != nil {
			return err
		}
		if d.Position() >= target {
			d.mu.Lock()
			d.pendingFrame = packet
			d.mu.Unlock()
			return nil
		}
	}
}

func (d *Demuxer) findSeekPointLocked(targetTicks int64) (seekPoint, bool) {
	if len(d.seekPoints) == 0 {
		if d.segmentStartSet && d.segmentDataStart >= 0 {
			return seekPoint{Timestamp: 0, Offset: d.segmentDataStart}, true
		}
		return seekPoint{}, false
	}
	i := sort.Search(len(d.seekPoints), func(i int) bool {
		return d.seekPoints[i].Timestamp > targetTicks
	})
	if i == 0 {
		return d.seekPoints[0], true
	}
	return d.seekPoints[i-1], true
}

func (d *Demuxer) addSeekPoint(timestamp, offset int64) {
	if timestamp < 0 || offset < 0 {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.seekPoints) == 0 {
		d.seekPoints = append(d.seekPoints, seekPoint{Timestamp: timestamp, Offset: offset})
		return
	}

	last := d.seekPoints[len(d.seekPoints)-1]
	if timestamp > last.Timestamp {
		d.seekPoints = append(d.seekPoints, seekPoint{Timestamp: timestamp, Offset: offset})
		return
	}
	if timestamp == last.Timestamp {
		if offset < last.Offset {
			d.seekPoints[len(d.seekPoints)-1].Offset = offset
		}
		return
	}

	i := sort.Search(len(d.seekPoints), func(i int) bool {
		return d.seekPoints[i].Timestamp >= timestamp
	})
	if i < len(d.seekPoints) && d.seekPoints[i].Timestamp == timestamp {
		if offset < d.seekPoints[i].Offset {
			d.seekPoints[i].Offset = offset
		}
		return
	}

	d.seekPoints = append(d.seekPoints, seekPoint{})
	copy(d.seekPoints[i+1:], d.seekPoints[i:])
	d.seekPoints[i] = seekPoint{Timestamp: timestamp, Offset: offset}
}

func (d *Demuxer) ProvideFrame() ([]byte, error) {
	d.opMu.Lock()
	defer d.opMu.Unlock()
	return d.provideFrameLocked()
}

func (d *Demuxer) provideFrameLocked() ([]byte, error) {
	d.mu.Lock()
	if len(d.pendingFrame) > 0 {
		frame := d.pendingFrame
		d.pendingFrame = nil
		d.mu.Unlock()
		return frame, nil
	}
	d.mu.Unlock()

	for {
		id, size, err := d.r.Next()
		if err == io.EOF {
			return nil, io.EOF
		}
		if err != nil {
			return nil, err
		}

		switch id {
		case ID_Segment:
			d.State = Segment{}
			if offset, err := d.r.Tell(); err == nil {
				d.mu.Lock()
				d.segmentDataStart = offset
				d.segmentStartSet = true
				d.mu.Unlock()
			}
			d.logDebug("Segment",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		// --- Segment/SeekHead

		case ID_SeekHead:
			sh := &SeekHead{}
			d.State.SeekHead = append(d.State.SeekHead, sh)
			d.logDebug("SeekHead",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_Seek:
			seekHead := d.State.SeekHead[len(d.State.SeekHead)-1]
			seekHead.Seek = append(seekHead.Seek, &Seek{})

			d.logDebug("Seek",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_SeekID:
			seekHead := d.State.SeekHead[len(d.State.SeekHead)-1]
			seek := seekHead.Seek[len(seekHead.Seek)-1]

			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			seek.ID = data

			d.logDebug("Seek ID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
				"hex", fmt.Sprintf("0x%02x", data),
			)

		case ID_SeekPosition:
			seekHead := d.State.SeekHead[len(d.State.SeekHead)-1]
			seek := seekHead.Seek[len(seekHead.Seek)-1]

			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			seek.Position = data

			d.logDebug("Seek Position",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		// --- Segment/Info

		case ID_Info:
			d.logDebug("Info",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_SegmentUUID:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.SegmentUUID = data

			d.logDebug("Info SegmentUUID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
				"hex", fmt.Sprintf("0x%02x", data),
			)

		case ID_SegmentFilename:
			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.SegmentFilename = data

			d.logDebug("Info SegmentFilename",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		case ID_PrevUUID:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.PrevUUID = data

			d.logDebug("Info PrevUUID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
				"hex", fmt.Sprintf("0x%02x", data),
			)

		case ID_PrevFilename:
			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.PrevFilename = data

			d.logDebug("Info PrevFilename",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		case ID_NextUUID:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.NextUUID = data

			d.logDebug("Info NextUUID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
				"hex", fmt.Sprintf("0x%02x", data),
			)

		case ID_NextFilename:
			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.NextFilename = data

			d.logDebug("Info NextFilename",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		case ID_SegmentFamily:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.SegmentFamily = data

			d.logDebug("Info SegmentFamily",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
				"hex", fmt.Sprintf("0x%02x", data),
			)

		case ID_ChapterTranslate:
			d.State.Info.ChapterTranslate = ChapterTranslate{}
			d.logDebug("Info ChapterTranslate",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_ChapterTranslateID:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.ChapterTranslate.ID = data

			d.logDebug("Info ChapterTranslate ID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
				"hex", fmt.Sprintf("0x%02x", data),
			)

		case ID_ChapterTranslateCodec:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.ChapterTranslate.Codec = data

			d.logDebug("Info ChapterTranslate Codec",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_ChapterTranslateEditionUID:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.ChapterTranslate.EditionUID = data

			d.logDebug("Info ChapterTranslate EditionUID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_TimestampScale:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.TimestampScale = data
			d.mu.Lock()
			d.timestampScale = data
			d.durationSet = false
			d.mu.Unlock()

			d.logDebug("Info TimestampScale",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_Duration:
			data, err := d.r.ReadFloat(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.Duration = data
			d.mu.Lock()
			d.durationSet = false
			d.mu.Unlock()

			d.logDebug("Info Duration",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%f", data),
			)

		case ID_DateUTC:
			data, err := d.r.ReadDate(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.DateUTC = data

			d.logDebug("Info DateUTC",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		case ID_Title:
			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.Title = data

			d.logDebug("Info Title",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		case ID_MuxingApp:
			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.MuxingApp = data

			d.logDebug("Info MuxingApp",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		case ID_WritingApp:
			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.State.Info.WritingApp = data

			d.logDebug("Info WritingApp",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		// --- Tracks

		case ID_Tracks:
			d.State.Tracks = Tracks{}
			d.logDebug("Tracks",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_TrackEntry:
			d.State.Tracks.TrackEntries = append(d.State.Tracks.TrackEntries, &TrackEntry{})
			d.logDebug(
				fmt.Sprintf("TrackEntry #%d", len(d.State.Tracks.TrackEntries)-1),
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_TrackNumber:
			trackEntry := d.State.Tracks.TrackEntries[len(d.State.Tracks.TrackEntries)-1]

			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			trackEntry.Number = data

			d.logDebug("TrackEntry Number",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_TrackUID:
			trackEntry := d.State.Tracks.TrackEntries[len(d.State.Tracks.TrackEntries)-1]

			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			trackEntry.UID = data

			d.logDebug("TrackEntry UID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_TrackType:
			//trackEntry := d.State.Tracks.TrackEntries[len(d.State.Tracks.TrackEntries)-1]

			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.logDebug("TrackEntry Type",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_Name:
			// trackEntry := d.State.Tracks.TrackEntries[len(d.State.Tracks.TrackEntries)-1]

			data, err := d.r.ReadString(size)
			if err != nil {
				return nil, err
			}
			d.logDebug("TrackEntry Name",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", data,
			)

		// --- Cluster

		case ID_Cluster:
			d.State.Cluster = Cluster{}
			if offset, err := d.r.Tell(); err == nil {
				d.mu.Lock()
				d.currentClusterOffset = offset
				d.mu.Unlock()
			}
			d.logDebug("Cluster",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_Timestamp:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.Timestamp = data
			d.mu.RLock()
			clusterOffset := d.currentClusterOffset
			d.mu.RUnlock()
			if clusterOffset >= 0 {
				d.addSeekPoint(int64(data), clusterOffset)
			}

			d.logDebug("Cluster Timestamp",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_Position:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.Position = data

			d.logDebug("Cluster Position",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_PrevSize:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.PrevSize = data

			d.logDebug("Cluster PrevSize",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_SimpleBlock:
			packet, err := d.readAndParseBlock(size)
			if err != nil {
				return nil, err
			}

			//d.logDebug("Cluster SimpleBlock",
			//	"ID", fmt.Sprintf("0x%02x", id),
			//	"size", size,
			//	"data", fmt.Sprintf("%d", packet),
			//)

			return packet, nil

		// --- BlockGroup

		case ID_BlockGroup:
			d.State.Cluster.BlockGroup = BlockGroup{}
			d.logDebug("BlockGroup",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_Block:
			packet, err := d.readAndParseBlock(size)
			if err != nil {
				return nil, err
			}

			d.logDebug("Cluster Block",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", packet),
			)

			return packet, nil

		// --- BlockAdditions

		case ID_BlockAdditions:
			d.State.Cluster.BlockGroup.BlockAdditions = BlockAdditions{}
			d.logDebug("BlockAdditions",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_BlockMore:
			d.State.Cluster.BlockGroup.BlockAdditions.BlockMore = BlockMore{}
			d.logDebug("BlockMore",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_BlockAdditional:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.BlockAdditions.BlockMore.BlockAdditional = data

			d.logDebug("Cluster BlockGroup BlockAdditions BlockMore BlockAdditional",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
			)

		case ID_BlockAddID:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.BlockAdditions.BlockMore.BlockAddID = data

			d.logDebug("Cluster BlockGroup BlockAdditions BlockMore BlockAddID",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_BlockDuration:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.BlockDuration = data

			d.logDebug("Cluster BlockGroup BlockDuration",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_ReferencePriority:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.ReferencePriority = data

			d.logDebug("Cluster BlockGroup ReferencePriority",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_ReferenceBlock:
			data, err := d.r.ReadInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.ReferenceBlock = data

			d.logDebug("Cluster BlockGroup ReferenceBlock",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_CodecState:
			data, err := d.r.ReadBinary(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.CodecState = data

			d.logDebug("Cluster BlockGroup CodecState",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%08b", data),
			)

		case ID_DiscardPadding:
			data, err := d.r.ReadInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cluster.BlockGroup.DiscardPadding = data

			d.logDebug("Cluster BlockGroup DiscardPadding",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		// --- Cues
		case ID_Cues:
			d.State.Cues = Cues{}
			d.logDebug("Cues",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			continue

		case ID_CuePoint:
			d.State.Cues.CuesPoint = CuesPoint{}
			d.logDebug("CuePoint")
			continue

		case ID_CueTime:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTime = data
			d.logDebug("CueTime", "data", fmt.Sprintf("%d", data))

		case ID_CueTrackPositions:
			cueTrackPosition := &CueTrackPosition{}
			d.State.Cues.CuesPoint.CueTrackPositions = append(d.State.Cues.CuesPoint.CueTrackPositions, cueTrackPosition)
			d.logDebug("CueTrackPositions", "size", size)
			continue

		case ID_CueTrack:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueTrack = data
			d.logDebug("CueTrack", "data", fmt.Sprintf("%d", data))

		case ID_CueClusterPosition:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueClusterPosition = data
			d.mu.RLock()
			segmentStart := d.segmentDataStart
			segmentStartSet := d.segmentStartSet
			cueTime := d.State.Cues.CuesPoint.CueTime
			d.mu.RUnlock()
			if segmentStartSet {
				d.addSeekPoint(int64(cueTime), segmentStart+int64(data))
			}
			d.logDebug("CueClusterPosition", "data", fmt.Sprintf("%d", data))

		case ID_CueRelativePosition:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueRelativePosition = data
			d.logDebug("CuePoint CueTrackPositions CueRelativePosition",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_CueDuration:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueDuration = data
			d.logDebug("CuePoint CueTrackPositions CueDuration",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_CueBlockNumber:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueBlockNumber = data
			d.logDebug("CuePoint CueTrackPositions CueBlockNumber",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_CueCodecState:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueCodecState = data
			d.logDebug("CuePoint CueTrackPositions CueCodecState",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
				"data", fmt.Sprintf("%d", data),
			)

		case ID_CueReference:
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueReference = CueReference{}
			d.logDebug("CuePoint CueTrackPositions CueReference")
			continue

		case ID_CueRefTime:
			data, err := d.r.ReadUInt(size)
			if err != nil {
				return nil, err
			}
			d.State.Cues.CuesPoint.CueTrackPositions[len(d.State.Cues.CuesPoint.CueTrackPositions)-1].CueReference.CueRefTime = data
			d.logDebug("CuePoint CueTrackPositions CueReference CueRefTime",
				"ID", fmt.Sprintf("0x%02x", id),
			)

		default:
			d.logDebug("Packet",
				"ID", fmt.Sprintf("0x%02x", id),
				"size", size,
			)
			err = d.r.Discard(size)
			if err != nil {
				return nil, err
			}
		}
	}

	return make([]byte, 0), nil
}

func (d *Demuxer) readAndParseBlock(size uint64) ([]byte, error) {
	bytesRead := 0

	_, n, err := d.r.ReadVInt(true)
	if err != nil {
		return nil, err
	}
	bytesRead += n

	timecodeData, err := d.r.ReadBinary(2)
	if err != nil {
		return nil, err
	}
	bytesRead += 2
	timecode := int16(binary.BigEndian.Uint16(timecodeData))

	flagData, err := d.r.ReadBinary(1)
	if err != nil {
		return nil, err
	}
	flags := flagData[0]
	bytesRead += 1

	lacing := (flags >> 1) & 0x03
	if lacing != 0 {
		return nil, fmt.Errorf("lacing not supported yet")
	}

	payloadSize := int(size) - bytesRead
	payload, err := d.r.ReadBinary(uint64(payloadSize))
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	clusterTimestamp := int64(d.State.Cluster.Timestamp)
	absolute := clusterTimestamp + int64(timecode)
	d.currentTimestamp = absolute
	d.mu.Unlock()

	return payload, nil
}
