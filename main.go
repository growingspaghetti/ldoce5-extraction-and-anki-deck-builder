package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type byteblockSettings struct {
	target               string
	assetType            string
	directoryBlockLen    int
	directoryParentBegin int
	directoryParentEnd   int
	fileBlockLen         int
	fileRawOffsetBegin   int
	fileRawOffsetEnd     int
	fileParentBegin      int
	fileParentEnd        int
}

type chunk struct {
	index, compressedSize, rawSize int
}

type directory struct {
	index, parent int
	name          string
}

type segmentfile struct {
	index, rawOffset, dir int
	name                  string
}

func compileDirectoryInfo(settings byteblockSettings) map[int]directory {
	nameFileContents, err := os.ReadFile(settings.target + "/dirs.skn/NAME.tda")
	if err != nil {
		panic(err)
	}
	names := bytes.Split(nameFileContents, []byte("\x00"))
	parentRelations, err := os.Open(settings.target + "/dirs.skn/dirs.dat")
	if err != nil {
		panic(err)
	}
	defer parentRelations.Close()

	directories := make(map[int]directory)
	parentInfo := make([]byte, settings.directoryBlockLen)
	for i := 0; ; i++ {
		read, err := parentRelations.Read(parentInfo)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if read != settings.directoryBlockLen {
			break
		}
		b, e := settings.directoryParentBegin, settings.directoryParentEnd
		directories[i] = directory{
			index:  i,
			parent: byteToInt(parentInfo[b:e]),
			name:   string(names[i]),
		}
	}
	return directories
}

func byteToInt(b []byte) int {
	switch len(b) {
	case 1:
		return int(b[0])
	case 2:
		return int(binary.LittleEndian.Uint16(b))
	case 3:
		u4 := make([]byte, 4)
		copy(u4, b)
		return int(binary.LittleEndian.Uint32(u4))
	case 4:
		return int(binary.LittleEndian.Uint32(b))
	default:
		panic(strconv.Itoa(len(b)) + "is illegal value")
	}
}

func compileSegmentFileInfo(settings byteblockSettings) []segmentfile {
	nameFileContents, err := os.ReadFile(settings.target + "/files.skn/NAME.tda")
	if err != nil {
		panic(err)
	}
	names := bytes.Split(nameFileContents, []byte("\x00"))
	segmentInfoTable, err := os.Open(settings.target + "/files.skn/files.dat")
	if err != nil {
		panic(err)
	}
	defer segmentInfoTable.Close()
	files := make([]segmentfile, 0)
	segmentInfo := make([]byte, settings.fileBlockLen)
	for i := 0; ; i++ {
		read, err := segmentInfoTable.Read(segmentInfo)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if read != settings.fileBlockLen {
			break
		}
		b1, e1 := settings.fileRawOffsetBegin, settings.fileRawOffsetEnd
		b2, e2 := settings.fileParentBegin, settings.fileParentEnd
		files = append(files, segmentfile{
			index:     i,
			rawOffset: byteToInt(segmentInfo[b1:e1]),
			dir:       byteToInt(segmentInfo[b2:e2]),
			name:      string(names[i]),
		})
	}
	return files
}

func compileChunkInfo(settings byteblockSettings) []chunk {
	chunkInfo, err := os.Open(settings.target + "/files.skn/CONTENT.tda.tdz")
	if err != nil {
		panic(err)
	}
	defer chunkInfo.Close()
	chunks := make([]chunk, 0)
	b := make([]byte, 8)
	for i := 0; ; i++ {
		read, err := chunkInfo.Read(b)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if read != 8 {
			break
		}
		zipSize := byteToInt(b[4:])
		rawSize := byteToInt(b[:4])
		chunks = append(chunks, chunk{
			index:          i,
			compressedSize: zipSize,
			rawSize:        rawSize,
		})
	}
	return chunks
}

func deflateFromChunk(chunks *os.File, chunk chunk) ([]byte, error) {
	zip := make([]byte, chunk.compressedSize)
	if _, err := chunks.Read(zip); err != nil {
		return nil, err
	}
	zipReader, err := zlib.NewReader(bytes.NewReader(zip))
	if err != nil {
		panic(err)
	}
	defer zipReader.Close()
	buf := bytes.NewBuffer(make([]byte, 0))
	if _, err := io.Copy(buf, zipReader); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extract(settings byteblockSettings) {
	chunkTable := compileChunkInfo(settings)
	segmentFiles := compileSegmentFileInfo(settings)
	directories := compileDirectoryInfo(settings)
	chunks, err := os.Open(settings.target + "/files.skn/CONTENT.tda")
	if err != nil {
		panic(err)
	}
	defer chunks.Close()

	accumulatoryBase := 0
	fileIndex := 0
	for _, chunk := range chunkTable {
		raw, err := deflateFromChunk(chunks, chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		for fileIndex < len(segmentFiles) && segmentFiles[fileIndex].rawOffset < accumulatoryBase+chunk.rawSize {
			relativeOffset := segmentFiles[fileIndex].rawOffset - accumulatoryBase
			data := fileData(segmentFiles, fileIndex, raw, relativeOffset)
			writeFile(segmentFiles[fileIndex], data, directories, settings.assetType)
			fileIndex++
		}
		accumulatoryBase += chunk.rawSize
	}
}

func fileData(segmentFiles []segmentfile, index int, chunk []byte, offset int) []byte {
	var data []byte
	if index == len(segmentFiles)-1 {
		data = chunk[offset:]
	} else {
		size := segmentFiles[index+1].rawOffset - segmentFiles[index].rawOffset
		data = chunk[offset : offset+size]
	}
	if strings.Contains(".xml.html.css", filepath.Ext(segmentFiles[index].name)) {
		data = bytes.Replace(data, []byte("\x00"), []byte{}, -1)
	}
	return data
}

func writeFile(segmentFile segmentfile, data []byte, directories map[int]directory, category string) {
	path := buildPath(directories, segmentFile)
	err := os.MkdirAll(category+path, 0777)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(category+path+"/"+segmentFile.name, data, 0644); err != nil {
		panic(err)
	}
}

func concatPath(dirs ...string) string {
	var sb strings.Builder
	for i := len(dirs) - 1; i >= 0; i-- {
		sb.WriteString("/")
		sb.WriteString(dirs[i])
	}
	return sb.String()
}

func buildPath(directories map[int]directory, root segmentfile) string {
	path := make([]string, 0)
	key := root.dir
	for {
		d, ok := directories[key]
		if !ok || key == 0 {
			return concatPath(path...)
		}
		path = append(path, d.name)
		key = d.parent
	}
}

var (
	dataPath string = "/home/ryoji/archive/ldoce5/ldoce5.data/"
)

func extractData() {
	if len(os.Args) > 1 {
		dataPath = os.Args[1]
		if !strings.HasSuffix(dataPath, "/") {
			dataPath += "/"
		}
	}

	fmt.Println("extracting title audio files")
	titleAudioSet := byteblockSettings{
		target:    dataPath + "gb_hwd_pron.skn",
		assetType: "media",
		// U24+USHORT+USHORT+USHORT=3+2+2+2
		directoryBlockLen:    9,
		directoryParentBegin: 7,
		directoryParentEnd:   9,
		// UBYTE+ULONG+U24+USHORT+USHORT+USHORT+USHORT=1+4+3+2+2+2+2
		fileBlockLen:       16,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   5,
		fileParentBegin:    14,
		fileParentEnd:      16,
	}
	extract(titleAudioSet)

	fmt.Println("extracting example audio files")
	exampleAudioSet := byteblockSettings{
		target:    dataPath + "exa_pron.skn",
		assetType: "media",
		// USHORT+UBYTE+U24+UBYTE=2+1+3+1
		directoryBlockLen:    7,
		directoryParentBegin: 6,
		directoryParentEnd:   7,
		// UBYTE+ULONG+U24+U24+U24+U24+UBYTE=1+4+3+3+3+3+1
		fileBlockLen:       18,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   5,
		fileParentBegin:    17,
		fileParentEnd:      18,
	}
	extract(exampleAudioSet)

	fmt.Println("extracting image files")
	imageSet := byteblockSettings{
		target:    dataPath + "picture.skn",
		assetType: "media",
		// USHORT+USHORT+USHORT+USHORT=2+2+2+2
		directoryBlockLen:    8,
		directoryParentBegin: 6,
		directoryParentEnd:   8,
		// UBYTE+ULONG+USHORT+USHORT+USHORT+USHORT+USHORT=1+4+2+2+2+2+2
		fileBlockLen:       15,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   5,
		fileParentBegin:    13,
		fileParentEnd:      15,
	}
	extract(imageSet)

	fmt.Println("extracting main text files")
	mainTextSet := byteblockSettings{
		target:    dataPath + "fs.skn",
		assetType: "text",
		// USHORT+USHORT+USHORT+USHORT=2+2+2+2
		directoryBlockLen:    8,
		directoryParentBegin: 7,
		directoryParentEnd:   8,
		// UBYTE+ULONG+U24+U24+USHORT+U24+USHORT=1+4+3+3+2+3+2
		fileBlockLen:       18,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   5,
		fileParentBegin:    16,
		fileParentEnd:      18,
	}
	extract(mainTextSet)

	fmt.Println("extracting the text for common errors")
	commonErrorSet := byteblockSettings{
		target:    dataPath + "common_errors.skn",
		assetType: "common-errors",
		// UBYTE+UBYTE+USHORT+UBYTE=1+1+2+1
		directoryBlockLen:    5,
		directoryParentBegin: 4,
		directoryParentEnd:   5,
		// UBYTE+U24+USHORT+USHORT+USHORT+USHORT+UBYTE=1+3+2+2+2+2+1
		fileBlockLen:       13,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   4,
		fileParentBegin:    12,
		fileParentEnd:      13,
	}
	extract(commonErrorSet)

	fmt.Println("extracting collocations")
	collocationSet := byteblockSettings{
		target:    dataPath + "collocations.skn",
		assetType: "collocations",
		// USHORT+USHORT+USHORT+USHORT=2+2+2+2
		directoryBlockLen:    8,
		directoryParentBegin: 6,
		directoryParentEnd:   8,
		// UBYTE+ULONG+U24+U24+USHORT+U24+USHORT=1+4+3+3+2+3+2
		fileBlockLen:       18,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   5,
		fileParentBegin:    16,
		fileParentEnd:      18,
	}
	extract(collocationSet)

	fmt.Println("extracting word lists")
	wordListSet := byteblockSettings{
		target:    dataPath + "word_lists.skn",
		assetType: "word-lists",
		// UBYTE+UBYTE+UBYTE+UBYTE=1+1+1+1
		directoryBlockLen:    4,
		directoryParentBegin: 3,
		directoryParentEnd:   4,
		// UBYTE+U24+UBYTE+UBYTE+UBYTE+UBYTE+UBYTE=1+3+1+1+1+1+1
		fileBlockLen:       9,
		fileRawOffsetBegin: 1,
		fileRawOffsetEnd:   4,
		fileParentBegin:    8,
		fileParentEnd:      9,
	}
	extract(wordListSet)
}

func main() {
	extractData()
	compileAnkiDeck()
}
