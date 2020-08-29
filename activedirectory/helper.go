package activedirectory

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// ErrObjectNotFound is custom error for object not found
var ErrObjectNotFound = errors.New("LDAP object not found")

const (
	accountDisabledFlag  uint64 = 2
	globalScopeFlag      uint64 = 2
	domainLocalScopeFlag uint64 = 4
	universalScopeFlag   uint64 = 8
	securityGroupFlag    uint64 = 2147483648
)

func ignoreCaseDiffSuppressor(k, old, new string, d *schema.ResourceData) bool {
	return strings.EqualFold(old, new)
}

func lowercaseHashString(v interface{}) int {
	return hashcode.String(strings.ToLower(v.(string)))
}

func validateAttributesJSON(val interface{}, key string) ([]string, []error) {
	var errs []error
	var warns []string
	dataMap := map[string][]string{}

	v := val.(string)
	err := json.Unmarshal([]byte(v), &dataMap)
	if err != nil {
		return warns, append(errs, fmt.Errorf(`%q must be valid json of map with string key and array of string as value ie. {key = ["value"]}, got: %s`, key, v))
	}
	for k, values := range dataMap {
		if len(values) == 0 {
			errs = append(errs, fmt.Errorf(`attributes values should not be empty. value of attribute %q got: %v`, k, values))
		}
	}
	return warns, errs
}

func normalizeAttributesJSON(val interface{}) string {
	dataMap := map[string][]string{}

	// Ignoring errors since attributes value is already validated by validateAttributesJSON
	_ = json.Unmarshal([]byte(val.(string)), &dataMap)

	// sort array before storing values in state since order in which
	// attributes values will be returned is not guaranteed
	// https://www.ietf.org/rfc/rfc2251.txt
	// 4.1.8. Attribute
	for _, v := range dataMap {
		sort.Strings(v)
	}
	ret, _ := json.Marshal(dataMap)

	return string(ret)
}

func validateDNString(c *ADClient, ou string) error {
	// validate OU Entry with to make sure its a full path
	errStr := ""
	if !strings.HasSuffix(strings.ToLower(ou), c.config.topDN) {
		errStr += fmt.Sprintf(`full ou path should end with top dn %q : `, c.config.topDN)
	}
	if _, err := ldap.ParseDN(strings.ToLower(ou)); err != nil {
		errStr += fmt.Sprintf("ou is not a valid DN err: %v", err)
	}
	if errStr != "" {
		return fmt.Errorf("error: %s, got: %s", errStr, ou)
	}
	return nil
}

func parseID(s string) string {
	var pStr string
	for i, char := range s {
		if i%2 == 0 {
			pStr += `\` + string(char)
			continue
		}
		pStr += string(char)
	}
	return pStr
}

func encodeSID(sid string) (string, error) {
	var rawSID []byte

	sid = strings.Replace(strings.ToLower(sid), "s-", "", 1)
	if len(sid) < 3 {
		return "", fmt.Errorf("sid string length is less then min allowed,  str:%s", sid)
	}
	subSec := strings.Split(sid, "-")

	// Encode revision
	r, err := strconv.Atoi(subSec[0])
	if err != nil {
		return "", fmt.Errorf("unable to revision string to int str:%s ,err:%w", subSec[1], err)
	}
	rawSID = append(rawSID, byte(r))

	// Encode Number of subsections
	rawSID = append(rawSID, byte(len(subSec)-2))

	// Encode Authority (six bytes, 48-bit number in big-endian format)
	a, err := strconv.ParseUint(subSec[1], 10, 48)
	if err != nil {
		return "", fmt.Errorf("unable to parse authority value ParseUint failed. err:%w", err)
	}
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, a)
	if err != nil {
		return "", fmt.Errorf("unable to write authority value binary.Write err:%w", err)
	}
	rawSID = append(rawSID, buf.Bytes()[2:]...)

	// Encode SubAuthorities (32-bit number in little-endian format)
	for i := 2; i < len(subSec); i++ {
		a, err := strconv.ParseUint(subSec[i], 10, 32)
		if err != nil {
			return "", fmt.Errorf("unable to parse authority value ParseUint failed. err:%w", err)
		}
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, a)
		if err != nil {
			return "", fmt.Errorf("unable to write authority value binary.Write err:%w", err)
		}
		rawSID = append(rawSID, buf.Bytes()[:4]...)
	}

	return fmt.Sprintf("%X", rawSID), nil
}

func decodeSID(sid []byte) (string, error) {
	// S-{Revision}-{Authority}-{SubAuthority1}-{SubAuthority2}...-{SubAuthorityN}
	sidStr := "S-"

	// validate sid array
	if len(sid) < 8 {
		return "", fmt.Errorf("unable to decode SID, sid byte is too small min req 8 byte given:%d", len(sid))
	}

	// revision
	sidStr += fmt.Sprintf("%X", sid[0])

	subAuthCount, err := strconv.ParseUint(fmt.Sprintf("%X", sid[1]), 16, 8)
	if err != nil {
		return "", fmt.Errorf("unable to decode SID, unable to get count of subauthority err:%w", err)
	}

	//byte(2-7) - 48 bit authority ([Big-Endian])
	buf := bytes.NewBuffer([]byte{0, 0})
	buf.Write(sid[2:8])
	authority := binary.BigEndian.Uint64(buf.Bytes())
	sidStr += "-" + fmt.Sprintf("%d", authority)

	//iterate all the sub-auths (four bytes, treated as a 32-bit number in little-endian format)
	var size uint64 = 4 //4 bytes for each sub auth
	var i uint64
	for i = 0; i < subAuthCount; i++ {
		from := 8 + (i * size)
		subAuth := binary.LittleEndian.Uint32(sid[from : from+size])
		sidStr += "-" + fmt.Sprintf("%d", subAuth)
	}
	return sidStr, nil
}

func decodeGUID(b []byte) (string, error) {
	if len(b) != 16 {
		return "", fmt.Errorf("size of raw guid is not 16 guid-len:%v", len(b))
	}
	x1 := binary.LittleEndian.Uint32(b[0:])
	x2 := binary.LittleEndian.Uint16(b[4:])
	x3 := binary.LittleEndian.Uint16(b[6:])
	x4 := binary.BigEndian.Uint16(b[8:])
	x5 := binary.BigEndian.Uint32(b[10:])
	x6 := binary.BigEndian.Uint16(b[14:])
	return fmt.Sprintf("%08X-%04X-%04X-%04X-%08X%04X", x1, x2, x3, x4, x5, x6), nil
}

func encodeGUID(guid string) (string, error) {
	b, err := hex.DecodeString(strings.ReplaceAll(guid, "-", ""))
	if err != nil {
		return "", fmt.Errorf("unable to decode guid string:%w", err)
	}
	if len(b) != 16 {
		return "", fmt.Errorf("size of decoded guid is not 16 guid-len:%v", len(b))
	}

	x1 := binary.LittleEndian.Uint32(b[0:])
	x2 := binary.LittleEndian.Uint16(b[4:])
	x3 := binary.LittleEndian.Uint16(b[6:])
	x4 := binary.BigEndian.Uint16(b[8:])
	x5 := binary.BigEndian.Uint32(b[10:])
	x6 := binary.BigEndian.Uint16(b[14:])

	return fmt.Sprintf("%08X%04X%04X%04X%08X%04X", x1, x2, x3, x4, x5, x6), nil
}

func isObjectEnabled(userAccountControl string) (bool, error) {

	uac, err := strconv.ParseUint(userAccountControl, 10, 64)
	if err != nil {
		return false, fmt.Errorf("unable to parse userAccountControl to uint %s", userAccountControl)
	}
	return !(uac&accountDisabledFlag != 0), nil
}

func setaccountDisabledFlag(userAccountControl string) (string, error) {

	uac, err := strconv.ParseUint(userAccountControl, 10, 64)
	if err != nil {
		return "", fmt.Errorf("unable to parse userAccountControl to uint %s", userAccountControl)
	}
	return strconv.FormatUint(uac|accountDisabledFlag, 10), nil
}

func unsetaccountDisabledFlag(userAccountControl string) (string, error) {

	uac, err := strconv.ParseUint(userAccountControl, 10, 64)
	if err != nil {
		return "", fmt.Errorf("unable to parse userAccountControl to uint %s", userAccountControl)
	}
	return strconv.FormatUint(uac&^accountDisabledFlag, 10), nil
}

// getModifiedAttributes will compare old and new attribute map and send difference in the as map added, replaced, deleted attributes
func getModifiedAttributes(oldAttrMap, newAttrMap map[string][]string) map[string][]string {
	replaced := map[string][]string{}

	// check of new added attributes
	for nName, nValues := range newAttrMap {
		if _, ok := oldAttrMap[nName]; !ok {
			replaced[nName] = nValues

		}
	}

	// check of Deleted attributes
	for oName := range oldAttrMap {
		if _, ok := newAttrMap[oName]; !ok {
			replaced[oName] = []string{}
		}
	}

	// check of updated/replaced attributes
	for oName, oValues := range oldAttrMap {
		if v, ok := newAttrMap[oName]; ok {
			if !compareAttrValues(oValues, v) {
				replaced[oName] = v
			}
		}
	}

	return replaced
}

func compareAttrValues(old, new []string) bool {
	if len(old) != len(new) {
		return false
	}
	sort.Strings(old)
	sort.Strings(new)
	return reflect.DeepEqual(old, new)
}

// getGroupTypeValue doesn't return error as both scope and type values are validated.
func getGroupTypeValue(gScope, gType string) string {
	var gTypeValue uint64

	// set group scope flags
	switch gScope {
	case groupScopeGlobal:
		gTypeValue = gTypeValue | globalScopeFlag
	case groupScopeDomainLocal:
		gTypeValue = gTypeValue | domainLocalScopeFlag
	case groupScopeUniversal:
		gTypeValue = gTypeValue | universalScopeFlag
	}

	// set security group flag
	if gType == groupTypeSecurity {
		return strconv.Itoa(int(gTypeValue) - int(securityGroupFlag))
	}

	// unset security group flag
	return strconv.Itoa(int(gTypeValue))
}

func getGroupTypeScope(value string) (string, string, error) {

	switch value {
	case "-2147483644":
		return groupScopeDomainLocal, groupTypeSecurity, nil
	case "-2147483640":
		return groupScopeUniversal, groupTypeSecurity, nil
	case "-2147483646":
		return groupScopeGlobal, groupTypeSecurity, nil
	case "2":
		return groupScopeGlobal, groupTypeDistribution, nil
	case "4":
		return groupScopeDomainLocal, groupTypeDistribution, nil
	case "8":
		return groupScopeUniversal, groupTypeDistribution, nil
	}

	return "", "", fmt.Errorf("unable to get scope or value for given group type value: %v", value)
}
