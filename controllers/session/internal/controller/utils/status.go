package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"

	sessionv1alpha1 "mr.telepresence/session/api/v1alpha1"
)

func HashStatus(status *sessionv1alpha1.SessionStatus) string {
	byteArray, _ := json.Marshal(*status)
	var hasher hash.Hash
	hasher = sha256.New()
	hasher.Reset()
	hasher.Write(byteArray)
	return hex.EncodeToString(hasher.Sum(nil))
}

func SetStatusHashAnnotation(hash string, session *sessionv1alpha1.Session) {
	if session.Annotations == nil {
		session.Annotations = make(map[string]string)
	}

	session.Annotations["statusHash"] = hash
}

func StatusHasChanged(oldStatusHash string, newStatusHash string) bool {
	return oldStatusHash != newStatusHash
}
