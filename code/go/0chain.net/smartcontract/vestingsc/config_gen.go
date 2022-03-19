package vestingsc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z Setting) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt(o, int(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Setting) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zb0001 int
		zb0001, bts, err = msgp.ReadIntBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		(*z) = Setting(zb0001)
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Setting) Msgsize() (s int) {
	s = msgp.IntSize
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *config) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "MinLock"
	o = append(o, 0x87, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
	o, err = z.MinLock.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MinLock")
		return
	}
	// string "MinDuration"
	o = append(o, 0xab, 0x4d, 0x69, 0x6e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendDuration(o, z.MinDuration)
	// string "MaxDuration"
	o = append(o, 0xab, 0x4d, 0x61, 0x78, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendDuration(o, z.MaxDuration)
	// string "MaxDestinations"
	o = append(o, 0xaf, 0x4d, 0x61, 0x78, 0x44, 0x65, 0x73, 0x74, 0x69, 0x6e, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73)
	o = msgp.AppendInt(o, z.MaxDestinations)
	// string "MaxDescriptionLength"
	o = append(o, 0xb4, 0x4d, 0x61, 0x78, 0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x4c, 0x65, 0x6e, 0x67, 0x74, 0x68)
	o = msgp.AppendInt(o, z.MaxDescriptionLength)
	// string "OwnerId"
	o = append(o, 0xa7, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x49, 0x64)
	o = msgp.AppendString(o, z.OwnerId)
	// string "Cost"
	o = append(o, 0xa4, 0x43, 0x6f, 0x73, 0x74)
	o = msgp.AppendMapHeader(o, uint32(len(z.Cost)))
	keys_za0001 := make([]string, 0, len(z.Cost))
	for k := range z.Cost {
		keys_za0001 = append(keys_za0001, k)
	}
	msgp.Sort(keys_za0001)
	for _, k := range keys_za0001 {
		za0002 := z.Cost[k]
		o = msgp.AppendString(o, k)
		o = msgp.AppendInt(o, za0002)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *config) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "MinLock":
			bts, err = z.MinLock.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinLock")
				return
			}
		case "MinDuration":
			z.MinDuration, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinDuration")
				return
			}
		case "MaxDuration":
			z.MaxDuration, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxDuration")
				return
			}
		case "MaxDestinations":
			z.MaxDestinations, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxDestinations")
				return
			}
		case "MaxDescriptionLength":
			z.MaxDescriptionLength, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxDescriptionLength")
				return
			}
		case "OwnerId":
			z.OwnerId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "OwnerId")
				return
			}
		case "Cost":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Cost")
				return
			}
			if z.Cost == nil {
				z.Cost = make(map[string]int, zb0002)
			} else if len(z.Cost) > 0 {
				for key := range z.Cost {
					delete(z.Cost, key)
				}
			}
			for zb0002 > 0 {
				var za0001 string
				var za0002 int
				zb0002--
				za0001, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Cost")
					return
				}
				za0002, bts, err = msgp.ReadIntBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Cost", za0001)
					return
				}
				z.Cost[za0001] = za0002
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *config) Msgsize() (s int) {
	s = 1 + 8 + z.MinLock.Msgsize() + 12 + msgp.DurationSize + 12 + msgp.DurationSize + 16 + msgp.IntSize + 21 + msgp.IntSize + 8 + msgp.StringPrefixSize + len(z.OwnerId) + 5 + msgp.MapHeaderSize
	if z.Cost != nil {
		for za0001, za0002 := range z.Cost {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + msgp.IntSize
		}
	}
	return
}