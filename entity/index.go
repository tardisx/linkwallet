package entity

type WordIndex struct {
	Word string `badgerhold:"index"`
	// Bitmap roaring.Bitmap
	Bitmap map[uint64]bool
}

// func (wi WordIndex) GobEncode() ([]byte, error) {

// 	bmBuf := new(bytes.Buffer)
// 	wi.Bitmap.WriteTo(bmBuf) // we omit error handling

// 	wordBytes := []byte(wi.Word)
// 	serialised := make([]byte, 4, 4)
// 	binary.BigEndian.PutUint32(serialised, uint32(len(wordBytes)))

// 	serialised = append(serialised, wordBytes...)
// 	serialised = append(serialised, bmBuf.Bytes()...)
// 	// log.Printf("serialised: %v", serialised)

// 	// log.Printf("serialised to %d bytes for word %w\n%#v", len(serialised), wi.Word, serialised)

// 	return serialised, nil
// }

// func (wi *WordIndex) GobDecode(b []byte) error {
// 	size := binary.BigEndian.Uint32(b[0:4])
// 	wi.Word = string(b[4 : size+4])
// 	// log.Printf("word is %s size was %d\n%v", wi.Word, size, b)

// 	bmBuf := bytes.NewReader(b[size+4:])

// 	wi.Bitmap = *roaring.New()
// 	_, err := wi.Bitmap.ReadFrom(bmBuf)
// 	// log.Printf("N: %d, err: %s", n, err)

// 	return err
// }
