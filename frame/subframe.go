package frame

type SubFrame struct {
	///Header *SubFrameHeader
	Block  interface{}
}

type SubFrameConstant struct {
	Value []byte
}

type SubFrameFixed struct {
	WarmUpSamples []byte
	Residual      []Residual
}

type SubFrameLpc struct {
	WarmUpSamples         []byte
	Precision             uint8
	ShiftNeeded           uint8
	PredictorCoefficients []byte
}

type SubFrameVerbatim struct {
	UnencodedSubblock []byte
}

type Residual struct {
	UsesRice2 bool
}

type Rice struct {
	PartitionOrder uint8
	Partitions     []RicePartition
}

type Rice2 struct {
	PartitionOrder uint8
	Partitions     []Rice2Partition
}

type RicePartition struct {
	EncodingParameter uint16
}

type Rice2Partition struct{}

/**
const (
	zeroPaddingMask  = 0x80
	subFrameTypeMask = 0x7E
)

subFrame := new(SubFrame)

c, err := buf.ReadByte()
if err != nil {
	return err
}

//Zero bit padding, to prevent sync-fooling string of 1s
if c&zeroPaddingMask != 0 {
	return nil, ErrIsNotNil
}

// Subframe type:
// 000000 : SUBFRAME_CONSTANT
// 000001 : SUBFRAME_VERBATIM
// 00001x : reserved
// 0001xx : reserved
// 001xxx : if(xxx <= 4) SUBFRAME_FIXED, xxx=order ; else reserved
// 01xxxx : reserved
// 1xxxxx : SUBFRAME_LPC, xxxxx=order-1

subFrame.Header.subFrameType = c & subFrameTypeMask
*/
