package live

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/wnjoon/go-yfinance/pkg/models"
)

// decodeBase64Message decodes a base64 encoded protobuf message.
func decodeBase64Message(base64Message string) (*models.PricingData, error) {
	decoded, err := base64.StdEncoding.DecodeString(base64Message)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}
	return decodeProtobuf(decoded)
}

// decodeProtobuf decodes a protobuf-encoded PricingData message.
// This is a manual decoder for the Yahoo Finance pricing.proto schema.
func decodeProtobuf(data []byte) (*models.PricingData, error) {
	pd := &models.PricingData{}
	reader := &protoReader{data: data, pos: 0}

	for reader.pos < len(reader.data) {
		fieldTag, wireType, err := reader.readTag()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		err = decodePricingField(pd, reader, fieldTag, wireType)

		if err != nil {
			return nil, fmt.Errorf("failed to decode field %d: %w", fieldTag, err)
		}
	}

	return pd, nil
}

var pricingFieldDecoders = map[int]func(*models.PricingData, *protoReader) error{
	1: func(pd *models.PricingData, r *protoReader) (err error) { pd.ID, err = r.readString(); return err },
	2: func(pd *models.PricingData, r *protoReader) (err error) { pd.Price, err = r.readFloat(); return err },
	3: func(pd *models.PricingData, r *protoReader) (err error) { pd.Time, err = r.readSint64(); return err },
	4: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.Currency, err = r.readString()
		return err
	},
	5: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.Exchange, err = r.readString()
		return err
	},
	6: func(pd *models.PricingData, r *protoReader) error {
		v, err := r.readVarint()
		pd.QuoteType = int32(v)
		return err
	},
	7: func(pd *models.PricingData, r *protoReader) error {
		v, err := r.readVarint()
		pd.MarketHours = int32(v)
		return err
	},
	8: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.ChangePercent, err = r.readFloat()
		return err
	},
	9: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.DayVolume, err = r.readSint64()
		return err
	},
	10: func(pd *models.PricingData, r *protoReader) (err error) { pd.DayHigh, err = r.readFloat(); return err },
	11: func(pd *models.PricingData, r *protoReader) (err error) { pd.DayLow, err = r.readFloat(); return err },
	12: func(pd *models.PricingData, r *protoReader) (err error) { pd.Change, err = r.readFloat(); return err },
	13: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.ShortName, err = r.readString()
		return err
	},
	14: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.ExpireDate, err = r.readSint64()
		return err
	},
	15: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.OpenPrice, err = r.readFloat()
		return err
	},
	16: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.PreviousClose, err = r.readFloat()
		return err
	},
	17: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.StrikePrice, err = r.readFloat()
		return err
	},
	18: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.UnderlyingSymbol, err = r.readString()
		return err
	},
	19: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.OpenInterest, err = r.readSint64()
		return err
	},
	20: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.OptionsType, err = r.readSint64()
		return err
	},
	21: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.MiniOption, err = r.readSint64()
		return err
	},
	22: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.LastSize, err = r.readSint64()
		return err
	},
	23: func(pd *models.PricingData, r *protoReader) (err error) { pd.Bid, err = r.readFloat(); return err },
	24: func(pd *models.PricingData, r *protoReader) (err error) { pd.BidSize, err = r.readSint64(); return err },
	25: func(pd *models.PricingData, r *protoReader) (err error) { pd.Ask, err = r.readFloat(); return err },
	26: func(pd *models.PricingData, r *protoReader) (err error) { pd.AskSize, err = r.readSint64(); return err },
	27: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.PriceHint, err = r.readSint64()
		return err
	},
	28: func(pd *models.PricingData, r *protoReader) (err error) { pd.Vol24Hr, err = r.readSint64(); return err },
	29: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.VolAllCurrencies, err = r.readSint64()
		return err
	},
	30: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.FromCurrency, err = r.readString()
		return err
	},
	31: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.LastMarket, err = r.readString()
		return err
	},
	32: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.CirculatingSupply, err = r.readDouble()
		return err
	},
	33: func(pd *models.PricingData, r *protoReader) (err error) {
		pd.MarketCap, err = r.readDouble()
		return err
	},
}

func decodePricingField(pd *models.PricingData, reader *protoReader, fieldTag, wireType int) error {
	if decode, ok := pricingFieldDecoders[fieldTag]; ok {
		return decode(pd, reader)
	}
	return reader.skipField(wireType)
}

// protoReader is a simple protobuf reader.
type protoReader struct {
	data []byte
	pos  int
}

// readTag reads a field tag and wire type.
func (r *protoReader) readTag() (fieldTag int, wireType int, err error) {
	if r.pos >= len(r.data) {
		return 0, 0, io.EOF
	}

	tag, err := r.readVarint()
	if err != nil {
		return 0, 0, err
	}

	fieldTag = int(tag >> 3)
	wireType = int(tag & 0x07)
	return fieldTag, wireType, nil
}

// readVarint reads a varint from the buffer.
func (r *protoReader) readVarint() (uint64, error) {
	var result uint64
	var shift uint

	for {
		if r.pos >= len(r.data) {
			return 0, io.ErrUnexpectedEOF
		}

		b := r.data[r.pos]
		r.pos++

		result |= uint64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}

	return result, nil
}

// readSint64 reads a signed varint (zigzag encoded).
func (r *protoReader) readSint64() (int64, error) {
	v, err := r.readVarint()
	if err != nil {
		return 0, err
	}
	// Zigzag decode
	return int64((v >> 1) ^ -(v & 1)), nil
}

// readString reads a length-delimited string.
func (r *protoReader) readString() (string, error) {
	length, err := r.readVarint()
	if err != nil {
		return "", err
	}

	if r.pos+int(length) > len(r.data) {
		return "", io.ErrUnexpectedEOF
	}

	s := string(r.data[r.pos : r.pos+int(length)])
	r.pos += int(length)
	return s, nil
}

// readFloat reads a 32-bit float.
func (r *protoReader) readFloat() (float32, error) {
	if r.pos+4 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}

	bits := binary.LittleEndian.Uint32(r.data[r.pos : r.pos+4])
	r.pos += 4
	return math.Float32frombits(bits), nil
}

// readDouble reads a 64-bit double.
func (r *protoReader) readDouble() (float64, error) {
	if r.pos+8 > len(r.data) {
		return 0, io.ErrUnexpectedEOF
	}

	bits := binary.LittleEndian.Uint64(r.data[r.pos : r.pos+8])
	r.pos += 8
	return math.Float64frombits(bits), nil
}

// skipField skips a field based on its wire type.
func (r *protoReader) skipField(wireType int) error {
	switch wireType {
	case 0: // Varint
		_, err := r.readVarint()
		return err
	case 1: // 64-bit
		if r.pos+8 > len(r.data) {
			return io.ErrUnexpectedEOF
		}
		r.pos += 8
		return nil
	case 2: // Length-delimited
		length, err := r.readVarint()
		if err != nil {
			return err
		}
		if r.pos+int(length) > len(r.data) {
			return io.ErrUnexpectedEOF
		}
		r.pos += int(length)
		return nil
	case 5: // 32-bit
		if r.pos+4 > len(r.data) {
			return io.ErrUnexpectedEOF
		}
		r.pos += 4
		return nil
	default:
		return fmt.Errorf("unknown wire type: %d", wireType)
	}
}
