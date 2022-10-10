package domain

type AuthCryptMsg struct {
	Protected  string `json:"protected"`
	Iv         string `json:"iv"`
	Ciphertext string `json:"ciphertext"`
	Tag        string `json:"tag"`
}

type Payload struct {
	Enc        string      `json:"enc"`
	Typ        string      `json:"typ"`
	Alg        string      `json:"alg"`
	Recipients []Recipient `json:"recipients"`
}

type Recipient struct {
	EncryptedKey string `json:"encrypted_key"`
	Header       Header `json:"header"`
}

type Header struct {
	Kid    string `json:"kid"`
	Iv     string `json:"iv"`
	Sender string `json:"sender"`
}
