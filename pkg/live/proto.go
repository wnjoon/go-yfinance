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

		switch fieldTag {
		case 1: // id (string)
			pd.ID, err = reader.readString()
		case 2: // price (float)
			pd.Price, err = reader.readFloat()
		case 3: // time (sint64)
			pd.Time, err = reader.readSint64()
		case 4: // currency (string)
			pd.Currency, err = reader.readString()
		case 5: // exchange (string)
			pd.Exchange, err = reader.readString()
		case 6: // quote_type (int32)
			v, e := reader.readVarint()
			pd.QuoteType = int32(v)
			err = e
		case 7: // market_hours (int32)
			v, e := reader.readVarint()
			pd.MarketHours = int32(v)
			err = e
		case 8: // change_percent (float)
			pd.ChangePercent, err = reader.readFloat()
		case 9: // day_volume (sint64)
			pd.DayVolume, err = reader.readSint64()
		case 10: // day_high (float)
			pd.DayHigh, err = reader.readFloat()
		case 11: // day_low (float)
			pd.DayLow, err = reader.readFloat()
		case 12: // change (float)
			pd.Change, err = reader.readFloat()
		case 13: // short_name (string)
			pd.ShortName, err = reader.readString()
		case 14: // expire_date (sint64)
			pd.ExpireDate, err = reader.readSint64()
		case 15: // open_price (float)
			pd.OpenPrice, err = reader.readFloat()
		case 16: // previous_close (float)
			pd.PreviousClose, err = reader.readFloat()
		case 17: // strike_price (float)
			pd.StrikePrice, err = reader.readFloat()
		case 18: // underlying_symbol (string)
			pd.UnderlyingSymbol, err = reader.readString()
		case 19: // open_interest (sint64)
			pd.OpenInterest, err = reader.readSint64()
		case 20: // options_type (sint64)
			pd.OptionsType, err = reader.readSint64()
		case 21: // mini_option (sint64)
			pd.MiniOption, err = reader.readSint64()
		case 22: // last_size (sint64)
			pd.LastSize, err = reader.readSint64()
		case 23: // bid (float)
			pd.Bid, err = reader.readFloat()
		case 24: // bid_size (sint64)
			pd.BidSize, err = reader.readSint64()
		case 25: // ask (float)
			pd.Ask, err = reader.readFloat()
		case 26: // ask_size (sint64)
			pd.AskSize, err = reader.readSint64()
		case 27: // price_hint (sint64)
			pd.PriceHint, err = reader.readSint64()
		case 28: // vol_24hr (sint64)
			pd.Vol24Hr, err = reader.readSint64()
		case 29: // vol_all_currencies (sint64)
			pd.VolAllCurrencies, err = reader.readSint64()
		case 30: // from_currency (string)
			pd.FromCurrency, err = reader.readString()
		case 31: // last_market (string)
			pd.LastMarket, err = reader.readString()
		case 32: // circulating_supply (double)
			pd.CirculatingSupply, err = reader.readDouble()
		case 33: // market_cap (double)
			pd.MarketCap, err = reader.readDouble()
		default:
			// Skip unknown field
			err = reader.skipField(wireType)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to decode field %d: %w", fieldTag, err)
		}
	}

	return pd, nil
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
