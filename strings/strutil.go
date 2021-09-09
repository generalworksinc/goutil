package gw_strings

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func CNullStr(a *string) string {
	if a == nil {
		return ""
	} else {
		return *a
	}
}
func Max(a string, b string) string {
	if a >= b {
		return a
	} else {
		return b
	}
}

func Min(a string, b string) string {
	if a <= b {
		return a
	} else {
		return b
	}
}

func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

func StreamToByte(stream io.Reader) []byte {
	if stream == nil {
		return []byte{}
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func StreamToString(stream io.Reader) string {
	if stream == nil {
		return ""
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.String()
}

var randSrc = rand.NewSource(time.Now().UnixNano())

const (
	rs6Letters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rs6LetterIdxBits = 6
	rs6LetterIdxMask = 1<<rs6LetterIdxBits - 1
	rs6LetterIdxMax  = 63 / rs6LetterIdxBits
)

// please set random seed before using.
// for example.
// rand.Seed(time.Now().UnixNano())
func RandString6(n int) string {
	b := make([]byte, n)
	cache, remain := randSrc.Int63(), rs6LetterIdxMax
	for i := n - 1; i >= 0; {
		if remain == 0 {
			cache, remain = randSrc.Int63(), rs6LetterIdxMax
		}
		idx := int(cache & rs6LetterIdxMask)
		if idx < len(rs6Letters) {
			b[i] = rs6Letters[idx]
			i--
		}
		cache >>= rs6LetterIdxBits
		remain--
	}
	return string(b)
}

func CompressStr(s string) (string, error) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	if _, err := gz.Write([]byte(s)); err != nil {
		return "", err
	}
	if err := gz.Flush(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	compressedStr := base64.StdEncoding.EncodeToString(b.Bytes())
	return compressedStr, nil
}
func DecompressStr(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	fmt.Println(data)
	rdata := bytes.NewReader(data)
	r, err := gzip.NewReader(rdata)
	if err != nil {
		return "", err
	}

	decompressedBytes, _ := ioutil.ReadAll(r)
	return string(decompressedBytes), nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

//マルチバイトを考慮したSubstring
func Substring(str string, start, length int) string {
	if start < 0 {
		panic("不正なindex指定です:" + strconv.Itoa(start))
	}
	if length <= 0 {
		return ""
	}
	r := []rune(str)
	if start+length > len(r) {
		return string(r[start:])
	} else {
		return string(r[start : start+length])
	}
}
func Substr(str string, start, end int) string {
	r := []rune(str)
	return string(r[start:end])
}
