package connections

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encrypt is used to encrypt the message
func Encrypt(key, text []byte) ([]byte, error) {

	//returns a new cipher for the key size
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//encode plain text to base64
	b := base64.StdEncoding.EncodeToString(text)

	ciphertext := make([]byte, aes.BlockSize+len(b))

	//initialization vector = first BlockSize bytes of ciphertext
	//IV's length = Block size
	iv := ciphertext[:aes.BlockSize]

	//iv = random bytes
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	//stream for cipher feedback mode
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))

	return ciphertext, nil
}

// Decrypt is used to decrypt the message
func Decrypt(key, text []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]

	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)

	data, err := base64.StdEncoding.DecodeString(string(text))

	if err != nil {
		return nil, err
	}

	return data, nil
}
