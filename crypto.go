package main

import (
	"crypto/aes"
	"crypto/cipher"
)

var key = []byte("DEADBEEFDEADBEEFDEADBEEFDEADBEEF")
var c, _ = aes.NewCipher(key)
var gcm, _ = cipher.NewGCM(c)
var nonce = make([]byte, gcm.NonceSize())

func EncryptString(str string) (encrypted string, err error) {

	text := []byte(str)

	encrypted = string(gcm.Seal(nonce, nonce, text, nil))

	return

}

func DecryptString(str string) (decrypted string, err error) {

	ciphertext := []byte(str)

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)

	decrypted = string(plaintext)

	return
}