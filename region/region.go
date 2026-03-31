package region

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.rtnl.ai/x/country"
)

// Region enumerates the clouds and regions that are available to Rotational in order to
// ensure region identification and serialization is as small a data type as possible.
// Region codes are generally broken into parts: the first digit represents the cloud,
// e.g. a region code that starts with 1 is Linode. The second series of three digits
// represents the country, e.g. USA is 840 in the ISO 3166 standard. The last three
// digits represents the zone of the datacenter, and is usually cloud specific.
//
// NOTE: this guide to the enumeration representation is generally about making the
// definition easier to see and parse; but the exact information of the region should
// be looked up using the RegionInfo struct.
type Region uint32

type Regions []Region

// Stores complete information about geographic metadata for compliance and provenance.
//
// Info can also be used to load the region information from environment variables
// with confire. The only required environment variable is the $REGION_INFO_ID from
// which all of the other region information can be derived. The other environment
// variables are optional and can be used to override the default values.
type Info struct {
	ID          Region           `json:"region" yaml:"region" msg:"region" env:"REGION_INFO_ID" split_words:"true" desc:"the r8l specific region identifier code (must be valid if not empty)"`
	Name        string           `json:"name" yaml:"name" msg:"name" env:"REGION_INFO_NAME" split_words:"true" desc:"the human readable name of the region"`
	CountryCode string           `json:"country_code" yaml:"country_code" msg:"country_code" env:"REGION_INFO_COUNTRY" split_words:"true" desc:"the ISO 3166-1 country code for the region"`
	Zone        string           `json:"zone" yaml:"zone" msg:"zone" env:"REGION_INFO_ZONE" split_words:"true" desc:"the zone of the datacenter for the region"`
	Cloud       string           `json:"cloud" yaml:"cloud" msg:"cloud" env:"REGION_INFO_CLOUD" split_words:"true" desc:"the cloud provider for the region"`
	Cluster     string           `json:"cluster" yaml:"cluster" msg:"cluster" env:"REGION_INFO_CLUSTER" split_words:"true" desc:"the r8l cluster name for the region"`
	country     *country.Country `json:"-" yaml:"-" msg:"-" env:"-"`
}

// Returns a list of all available regions known by the Rotational system.
func List() Regions {
	regions := make(Regions, 0, len(regionNames))
	for r := range regionNames {
		if r == 0 {
			continue
		}
		regions = append(regions, Region(r))
	}
	return regions
}

// Parse a region from a string or integer representation.
func Parse(s interface{}) (_ Region, err error) {
	switch v := s.(type) {
	case string:
		if isDigits(v) {
			var n uint64
			if n, err = strconv.ParseUint(v, 10, 32); err != nil {
				return 0, err
			}
			return Parse(uint32(n))
		}

		v = normalize(v)
		if region, ok := regionValues[v]; ok {
			return Region(region), nil
		}
		return UNKNOWN, fmt.Errorf("unknown region: %q", v)
	case json.Number:
		var n int64
		if n, err = v.Int64(); err != nil {
			return UNKNOWN, err
		}
		return Parse(uint32(n))
	case float32:
		return Parse(uint32(v))
	case float64:
		return Parse(uint32(v))
	case int:
		return Parse(uint32(v))
	case int32:
		return Parse(uint32(v))
	case int64:
		return Parse(uint32(v))
	case uint:
		return Parse(uint32(v))
	case uint32:
		if _, ok := regionNames[v]; ok {
			return Region(v), nil
		}
		return UNKNOWN, fmt.Errorf("unknown region: %d", v)
	case uint64:
		return Parse(uint32(v))
	default:
		return UNKNOWN, fmt.Errorf("cannot parse region from %T", v)
	}
}

func (r Region) String() string {
	if name, ok := regionNames[uint32(r)]; ok {
		return denormalize(name)
	}
	return "unknown"
}

func (r Region) Info() *Info {
	info := &Info{
		ID:   r,
		Name: r.String(),
	}

	codestr := fmt.Sprintf("%07d", r)
	cc := codestr[1:4]
	info.country, _ = country.Code(cc)

	if info.country != nil {
		info.CountryCode = info.country.Alpha2
	}

	parts := strings.Split(normalize(info.Name), "_")
	if len(parts) > 2 {
		info.Cloud = parts[0]
		info.Zone = parts[len(parts)-1]
	}

	return info
}

func (r *Info) Validate() (err error) {
	if r.ID != UNKNOWN {
		info := r.ID.Info()
		if r.Name == "" {
			r.Name = info.Name
		}

		if r.CountryCode == "" {
			r.CountryCode = info.CountryCode
			r.country = info.country
		} else {
			if r.country, err = country.Lookup(r.CountryCode); err != nil {
				return fmt.Errorf("unknown country code %q", r.CountryCode)
			}
		}

		if r.Zone == "" {
			r.Zone = info.Zone
		}

		if r.Cloud == "" {
			r.Cloud = info.Cloud
		}
	}

	return nil
}

func (r *Info) Country() *country.Country {
	if r.country == nil {
		r.country, _ = country.Lookup(r.CountryCode)
	}
	return r.country
}

//============================================================================
// Serialization
//============================================================================

func (r Region) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *Region) UnmarshalJSON(data []byte) (err error) {
	var s interface{}
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if *r, err = Parse(s); err != nil {
		return err
	}
	return nil
}

func (r Region) MarshalBinary() ([]byte, error) {
	data := make([]byte, binary.MaxVarintLen32)
	i := binary.PutUvarint(data, uint64(r))
	return data[:i], nil
}

func (r *Region) UnmarshalBinary(data []byte) (err error) {
	if len(data) > binary.MaxVarintLen32 {
		data = data[:binary.MaxVarintLen32]
	}

	n, _ := binary.Uvarint(data)
	*r, err = Parse(uint32(n))
	return err
}

func (r *Region) Decode(s string) (err error) {
	*r, err = Parse(s)
	return err
}

//============================================================================
// Helper Methods
//============================================================================

func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func normalize(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func denormalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
