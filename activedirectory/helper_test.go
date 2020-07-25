package activedirectory

import (
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func Test_decodeGUID(t *testing.T) {
	type args struct {
		b string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "1", args: args{``}, want: "", wantErr: true},
		{name: "2", args: args{`52234B1739997B4A996C4F605D078DF5`}, want: "174b2352-9939-4a7b-996c-4f605d078df5", wantErr: false},
		{name: "3", args: args{`DD75106C51B0254E87823494F8386189`}, want: "6c1075dd-b051-4e25-8782-3494f8386189", wantErr: false},
		{name: "4", args: args{`5DEAEDB9942F784D87B86B75AF7017D6`}, want: "b9edea5d-2f94-4d78-87b8-6b75af7017d6", wantErr: false},
		{name: "5", args: args{`99FAFD1041A4814E88CDE98286A7B3CF`}, want: "10fdfa99-a441-4e81-88cd-e98286a7b3cf", wantErr: false},
		{name: "6", args: args{`75F19289BCE7134DB3E125B3DA9AC7B6`}, want: "8992f175-e7bc-4d13-b3e1-25b3da9ac7b6", wantErr: false},
		{name: "7", args: args{`39E99E00987EE34D861548DC9A866E9B`}, want: "009ee939-7e98-4de3-8615-48dc9a866e9b", wantErr: false},
		{name: "8", args: args{`42C7817C44125E4DA68CF39A5E08E5D7`}, want: "7c81c742-1244-4d5e-a68c-f39a5e08e5d7", wantErr: false},
		{name: "9", args: args{`18CC5C3F907D0F47AD2BAE6722E31953`}, want: "3f5ccc18-7d90-470f-ad2b-ae6722e31953", wantErr: false},
		{name: "10", args: args{`460F91BC9D4B9541AC14DAA1AABA26FD`}, want: "bc910f46-4b9d-4195-ac14-daa1aaba26fd", wantErr: false},
		{name: "11", args: args{`66244B34803A4840B266293B239C8B88`}, want: "344b2466-3a80-4048-b266-293b239c8b88", wantErr: false},
		{name: "12", args: args{`00A8C6967B5307479AD70283F86A833B`}, want: "96c6a800-537b-4707-9ad7-0283f86a833b", wantErr: false},
		{name: "13", args: args{`D33972939DF0BA4193425CE61F856181`}, want: "937239d3-f09d-41ba-9342-5ce61f856181", wantErr: false},
		{name: "14", args: args{`FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF`}, want: "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", wantErr: false},
		{name: "15", args: args{`00000000000000000000000000000000`}, want: "00000000-0000-0000-0000-000000000000", wantErr: false},
		{name: "16", args: args{`000000000000000000000000000000`}, want: "", wantErr: true},
	}
	for _, tt := range tests {
		b, err := hex.DecodeString(tt.args.b)
		if err != nil {
			t.Errorf("DecodeString() name=%s error = %v", tt.name, err)
		}
		got, err := decodeGUID(b)
		if (err != nil) != tt.wantErr {
			t.Errorf("decodeGUID() name=%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if !strings.EqualFold(got, tt.want) {
			t.Errorf("decodeGUID() name=%s Got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_encodeGUID(t *testing.T) {
	type args struct {
		guid string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "1", args: args{``}, want: "", wantErr: true},
		{name: "2", args: args{"174b2352-9939-4a7b-996c-4f605d078df5"}, want: `52234B1739997B4A996C4F605D078DF5`, wantErr: false},
		{name: "3", args: args{"6c1075dd-b051-4e25-8782-3494f8386189"}, want: `DD75106C51B0254E87823494F8386189`, wantErr: false},
		{name: "4", args: args{"b9edea5d-2f94-4d78-87b8-6b75af7017d6"}, want: `5DEAEDB9942F784D87B86B75AF7017D6`, wantErr: false},
		{name: "5", args: args{"10fdfa99-a441-4e81-88cd-e98286a7b3cf"}, want: `99FAFD1041A4814E88CDE98286A7B3CF`, wantErr: false},
		{name: "6", args: args{"8992f175-e7bc-4d13-b3e1-25b3da9ac7b6"}, want: `75F19289BCE7134DB3E125B3DA9AC7B6`, wantErr: false},
		{name: "7", args: args{"009ee939-7e98-4de3-8615-48dc9a866e9b"}, want: `39E99E00987EE34D861548DC9A866E9B`, wantErr: false},
		{name: "8", args: args{"7c81c742-1244-4d5e-a68c-f39a5e08e5d7"}, want: `42C7817C44125E4DA68CF39A5E08E5D7`, wantErr: false},
		{name: "9", args: args{"3f5ccc18-7d90-470f-ad2b-ae6722e31953"}, want: `18CC5C3F907D0F47AD2BAE6722E31953`, wantErr: false},
		{name: "10", args: args{"bc910f46-4b9d-4195-ac14-daa1aaba26fd"}, want: `460F91BC9D4B9541AC14DAA1AABA26FD`, wantErr: false},
		{name: "11", args: args{"344b2466-3a80-4048-b266-293b239c8b88"}, want: `66244B34803A4840B266293B239C8B88`, wantErr: false},
		{name: "12", args: args{"96c6a800-537b-4707-9ad7-0283f86a833b"}, want: `00A8C6967B5307479AD70283F86A833B`, wantErr: false},
		{name: "13", args: args{"937239d3-f09d-41ba-9342-5ce61f856181"}, want: `D33972939DF0BA4193425CE61F856181`, wantErr: false},
		{name: "14", args: args{"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF"}, want: `FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF`, wantErr: false},
		{name: "15", args: args{"00000000-0000-0000-0000-000000000000"}, want: `00000000000000000000000000000000`, wantErr: false},
		{name: "16", args: args{"344b2466-3a80-4048-b266-293b239c8b"}, want: ``, wantErr: true},
		{name: "17", args: args{"96c6a800-537b-4707-9a-0283f86a833b"}, want: ``, wantErr: true},
	}
	for _, tt := range tests {
		got, err := encodeGUID(tt.args.guid)
		if (err != nil) != tt.wantErr {
			t.Errorf("encodeGUID() name = %v error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("encodeGUID() name = %v got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_decodeSID(t *testing.T) {
	type args struct {
		b string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "1", args: args{``}, want: "", wantErr: true},
		{name: "2", args: args{`0105000000000005150000005054D6BDA93719ED5365340D85040000`}, want: "S-1-5-21-3184940112-3977852841-221537619-1157", wantErr: false},
		{name: "3", args: args{`0105000000000005150000005054D6BDA93719ED5365340D52040000`}, want: "S-1-5-21-3184940112-3977852841-221537619-1106", wantErr: false},
		{name: "4", args: args{`0105000000000005150000005054D6BDA93719ED5365340D50040000`}, want: "S-1-5-21-3184940112-3977852841-221537619-1104", wantErr: false},
		{name: "5", args: args{`0105000000000005FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF`}, want: "S-1-5-4294967295-4294967295-4294967295-4294967295-4294967295", wantErr: false},
		{name: "6", args: args{`0103000000000005FFFFFFFFFFFFFFFFFFFFFFFF`}, want: "S-1-5-4294967295-4294967295-4294967295", wantErr: false},
		{name: "7", args: args{`01050000000000050000000000000000000000000000000000000000`}, want: "S-1-5-0-0-0-0-0", wantErr: false},
		{name: "8", args: args{`0100000000000005`}, want: "S-1-5", wantErr: false},
		{
			name:    "9",
			args:    args{`010FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF`},
			want:    "S-1-281474976710655-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		b, err := hex.DecodeString(tt.args.b)
		if err != nil {
			t.Errorf("DecodeString() name = %s error = %v", tt.name, err)
		}
		got, err := decodeSID(b)
		if (err != nil) != tt.wantErr {
			t.Errorf("decodeGUID() name = %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if !strings.EqualFold(got, tt.want) {
			t.Errorf("decodeGUID() name = %s Got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_encodeSID(t *testing.T) {
	type args struct {
		sid string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "1", args: args{``}, want: "", wantErr: true},
		{name: "2", args: args{"S-1-5-21-3184940112-3977852841-221537619-1157"}, want: `0105000000000005150000005054D6BDA93719ED5365340D85040000`, wantErr: false},
		{name: "3", args: args{"S-1-5-21-3184940112-3977852841-221537619-1106"}, want: `0105000000000005150000005054D6BDA93719ED5365340D52040000`, wantErr: false},
		{name: "4", args: args{"S-1-5-21-3184940112-3977852841-221537619-1104"}, want: `0105000000000005150000005054D6BDA93719ED5365340D50040000`, wantErr: false},
		{name: "5", args: args{"S-1-5-4294967295-4294967295-4294967295-4294967295-4294967295"}, want: `0105000000000005FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF`, wantErr: false},
		{name: "6", args: args{"S-1-5-4294967295-4294967295-4294967295"}, want: `0103000000000005FFFFFFFFFFFFFFFFFFFFFFFF`, wantErr: false},
		{name: "7", args: args{"S-1-5-0-0-0-0-0"}, want: `01050000000000050000000000000000000000000000000000000000`, wantErr: false},
		{name: "8", args: args{`S-1-5`}, want: "0100000000000005", wantErr: false},
		{
			name:    "9",
			args:    args{`S-1-281474976710655-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295-4294967295`},
			want:    "010FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := encodeSID(tt.args.sid)
		if (err != nil) != tt.wantErr {
			t.Errorf("encodeSID() name = %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("encodeSID() name = %s Got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_isObjectEnabled(t *testing.T) {
	type args struct {
		userAccountControl string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "1", args: args{"66050"}, want: false, wantErr: false},
		{name: "2", args: args{"514"}, want: false, wantErr: false},
		{name: "3", args: args{"66048"}, want: true, wantErr: false},
		{name: "4", args: args{"4096"}, want: true, wantErr: false},
		{name: "5", args: args{"512"}, want: true, wantErr: false},
		{name: "6", args: args{"532480"}, want: true, wantErr: false},
		{name: "7", args: args{"4130"}, want: false, wantErr: false},
		{name: "8", args: args{"4128"}, want: true, wantErr: false},
	}
	for _, tt := range tests {
		got, err := isObjectEnabled(tt.args.userAccountControl)
		if (err != nil) != tt.wantErr {
			t.Errorf("isObjectEnabled() name= %v error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("isObjectEnabled() name= %v Got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_setaccountDisabledFlag(t *testing.T) {
	type args struct {
		userAccountControl string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "1", args: args{"66050"}, want: "66050", wantErr: false},
		{name: "2", args: args{"514"}, want: "514", wantErr: false},
		{name: "3", args: args{"66048"}, want: "66050", wantErr: false},
		{name: "4", args: args{"4096"}, want: "4098", wantErr: false},
		{name: "5", args: args{"512"}, want: "514", wantErr: false},
		{name: "6", args: args{"532480"}, want: "532482", wantErr: false},
		{name: "7", args: args{"4130"}, want: "4130", wantErr: false},
		{name: "8", args: args{"4128"}, want: "4130", wantErr: false},
	}
	for _, tt := range tests {
		got, err := setaccountDisabledFlag(tt.args.userAccountControl)
		if (err != nil) != tt.wantErr {
			t.Errorf("setaccountDisabledFlag() name = %s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("setaccountDisabledFlag() name = %s Got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_getModifiedAttributes(t *testing.T) {
	type args struct {
		oldAttrMap map[string][]string
		newAttrMap map[string][]string
	}
	tests := []struct {
		name     string
		args     args
		added    map[string][]string
		replaced map[string][]string
		deleted  map[string][]string
	}{
		{
			name: "1",
			args: args{
				oldAttrMap: map[string][]string{
					"department":       {"IT TF"},
					"departmentNumber": {"testing1", "testing2"},
					"countryCode":      {"2"},
					"employeeNumber":   {"234"},
					"test":             {"123"},
					"nochange":         {"nochange"},
				},
				newAttrMap: map[string][]string{
					"department":       {"IT TF Update"},
					"departmentNumber": {"testing2", "testing1", "testing3"},
					"company":          {"home"},
					"test":             {"345"},
					"nochange":         {"nochange"},
				},
			},
			replaced: map[string][]string{
				"company":          {"home"},
				"test":             {"345"},
				"countryCode":      {},
				"employeeNumber":   {},
				"department":       {"IT TF Update"},
				"departmentNumber": {"testing2", "testing1", "testing3"},
			},
		}, {
			name: "2",
			args: args{
				oldAttrMap: map[string][]string{
					"department":       {"IT TF"},
					"departmentNumber": {"testing1", "testing2"},
				},
				newAttrMap: map[string][]string{},
			},
			replaced: map[string][]string{
				"department":       {},
				"departmentNumber": {},
			},
		}, {
			name: "3",
			args: args{
				oldAttrMap: map[string][]string{},
				newAttrMap: map[string][]string{
					"department":       {"IT TF"},
					"departmentNumber": {"testing1", "testing2"},
				},
			},
			replaced: map[string][]string{
				"department":       {"IT TF"},
				"departmentNumber": {"testing1", "testing2"},
			},
		}, {
			name: "4",
			args: args{
				oldAttrMap: map[string][]string{},
				newAttrMap: map[string][]string{},
			},
			added:    map[string][]string{},
			replaced: map[string][]string{},
			deleted:  map[string][]string{},
		},
	}
	for _, tt := range tests {
		got := getModifiedAttributes(tt.args.oldAttrMap, tt.args.newAttrMap)
		if !reflect.DeepEqual(got, tt.replaced) {
			t.Errorf("getModifiedAttributes() name = %s - Replaced - got = %#v, want %#v", tt.name, got, tt.replaced)
		}
	}
}

func Test_getGroupTypeValue(t *testing.T) {
	type args struct {
		gScope string
		gType  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "1", args: args{groupScopeDomainLocal, groupTypeSecurity}, want: "-2147483644"},
		{name: "2", args: args{groupScopeUniversal, groupTypeSecurity}, want: "-2147483640"},
		{name: "3", args: args{groupScopeGlobal, groupTypeSecurity}, want: "-2147483646"},
		{name: "4", args: args{groupScopeDomainLocal, groupTypeDistribution}, want: "4"},
		{name: "5", args: args{groupScopeUniversal, groupTypeDistribution}, want: "8"},
	}
	for _, tt := range tests {
		if got := getGroupTypeValue(tt.args.gScope, tt.args.gType); got != tt.want {
			t.Errorf("getGroupTypeValue() name = %s got = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_getGroupTypeScope(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{name: "1", args: args{"-2147483644"}, want: groupScopeDomainLocal, want1: groupTypeSecurity, wantErr: false},
		{name: "2", args: args{"-2147483640"}, want: groupScopeUniversal, want1: groupTypeSecurity, wantErr: false},
		{name: "3", args: args{"-2147483646"}, want: groupScopeGlobal, want1: groupTypeSecurity, wantErr: false},
		{name: "4", args: args{"2"}, want: groupScopeGlobal, want1: groupTypeDistribution, wantErr: false},
		{name: "5", args: args{"4"}, want: groupScopeDomainLocal, want1: groupTypeDistribution, wantErr: false},
		{name: "6", args: args{"8"}, want: groupScopeUniversal, want1: groupTypeDistribution, wantErr: false},
		{name: "7", args: args{"100"}, want: "", want1: "", wantErr: true},
	}
	for _, tt := range tests {
		got, got1, err := getGroupTypeScope(tt.args.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("getGroupTypeScope() name=%s error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if got != tt.want {
			t.Errorf("getGroupTypeScope() name=%s got = %v, want %v", tt.name, got, tt.want)
		}
		if got1 != tt.want1 {
			t.Errorf("getGroupTypeScope() name=%s got1 = %v, want %v", tt.name, got1, tt.want1)
		}
	}
}
