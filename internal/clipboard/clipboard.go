package clipboard

type Clipboard struct {
	Name         string `json:"name"`
	DataType     string `json:"type"`
	Data         string `json:"data"`
	IsEncrypted  bool   `json:"is_encrypted"`
	PasswordHash string `json:"-"`
	Salt         string `json:"-"`
	Nonce        string `json:"-"`
}

// NewClipboard creates a new clipboard with the given name, data type, and data.
func NewClipboard(name, dataType, data string) *Clipboard {
	return &Clipboard{
		Name:        name,
		DataType:    dataType,
		Data:        data,
		IsEncrypted: false,
	}
}
