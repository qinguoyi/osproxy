package base

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
)

func decode(message string) string {
	h := hmac.New(sha256.New, []byte(utils.EncryKey))
	h.Write([]byte(message))
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func GenUploadSignature(uid, date string, expire int, signature string) string {
	standardizedQueryString := fmt.Sprintf(
		"uid=%s&date=%s&expire=%d&signature=%s",
		uid,
		date,
		expire,
		signature,
	)
	return standardizedQueryString
}

func CheckUploadSignature(date, expire, signature string) bool {
	decodeRes := decode(fmt.Sprintf("%s-%s", date, expire))
	return decodeRes == signature
}

func GenDownloadSignature(uid int64, srcName, bucket, objectName, expire, date, signature string) string {
	standardizedQueryString := fmt.Sprintf(
		"uid=%d&name=%s&date=%s&expire=%s&bucket=%s&object=%s&signature=%s",
		uid,
		srcName,
		date,
		expire,
		bucket,
		objectName,
		signature,
	)
	return standardizedQueryString
}

func CheckDownloadSignature(date, expire, bucket, objectName, signature string) bool {
	decodeRes := decode(fmt.Sprintf("%s-%s-%s-%s", date, expire, bucket, objectName))
	return decodeRes == signature
}
