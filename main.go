package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

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

func compileDirectoryInfo() map[int]directory {
	nameFile, err := os.ReadFile("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/dirs.skn/NAME.tda")
	if err != nil {
		panic(err)
	}
	names := bytes.Split(nameFile, []byte("\x00"))
	parentRelations, err := os.Open("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/dirs.skn/dirs.dat")
	if err != nil {
		panic(err)
	}
	defer parentRelations.Close()

	directories := make(map[int]directory)
	parentInfo := make([]byte, 9)
	for i := 0; ; i++ {
		read, err := parentRelations.Read(parentInfo)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if read != 9 {
			break
		}
		directories[i] = directory{
			index:  i,
			parent: int(binary.LittleEndian.Uint16(parentInfo[7:9])),
			name:   string(names[i]),
		}
	}
	return directories
}

func compileSegmentFileInfo() []segmentfile {
	nameFile, err := os.ReadFile("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/NAME.tda")
	if err != nil {
		panic(err)
	}
	names := bytes.Split(nameFile, []byte("\x00"))
	segmentInfoTable, err := os.Open("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/files.skn/files.dat")
	if err != nil {
		panic(err)
	}
	defer segmentInfoTable.Close()
	files := make([]segmentfile, 0)
	segmentInfo := make([]byte, 16)
	for i := 0; ; i++ {
		read, err := segmentInfoTable.Read(segmentInfo)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if read != 16 {
			break
		}
		files = append(files, segmentfile{
			index:     i,
			rawOffset: int(binary.LittleEndian.Uint32(segmentInfo[1:5])),
			dir:       int(binary.LittleEndian.Uint16(segmentInfo[14:16])),
			name:      string(names[i]),
		})
	}
	return files
}

func compileChunkInfo() []chunk {
	chunkInfo, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/CONTENT.tda.tdz")
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
		zipSize := uint32(binary.LittleEndian.Uint32(b[4:]))
		rawSize := uint32(binary.LittleEndian.Uint32(b[:4]))
		chunks = append(chunks, chunk{
			index:          i,
			compressedSize: int(zipSize),
			rawSize:        int(rawSize),
		})
	}
	return chunks
}

func extractTitleAudio() {
	chunkTable := compileChunkInfo()
	segmentFiles := compileSegmentFileInfo()
	directories := compileDirectoryInfo()

	chunks, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/CONTENT.tda")
	if err != nil {
		panic(err)
	}
	defer chunks.Close()

	accumulatoryBase := 0
	fileIndex := 0
	for i, chunk := range chunkTable {
		if i > 5 {
			break
		}
		zip := make([]byte, chunk.compressedSize)
		if _, err := chunks.Read(zip); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		zipReader, err := zlib.NewReader(bytes.NewReader(zip))
		if err != nil {
			panic(err)
		}
		defer zipReader.Close()
		buf := bytes.NewBuffer(make([]byte, 0))
		if _, err := io.Copy(buf, zipReader); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		raw := buf.Bytes()
		for segmentFiles[fileIndex].rawOffset < accumulatoryBase+chunk.rawSize {
			relativeOffset := segmentFiles[fileIndex].rawOffset - accumulatoryBase
			data := fileData(segmentFiles, fileIndex, raw, relativeOffset)
			writeFile(segmentFiles[fileIndex], data, directories)
			fileIndex++
		}
		accumulatoryBase += chunk.rawSize
	}
}

func fileData(segmentFiles []segmentfile, index int, chunk []byte, offset int) []byte {
	if index == len(segmentFiles)-1 {
		return chunk[offset:]
	} else {
		size := segmentFiles[index+1].rawOffset - segmentFiles[index].rawOffset
		return chunk[offset : offset+size]
	}
}

func writeFile(segmentFile segmentfile, data []byte, directories map[int]directory) {
	path := buildPath(directories, segmentFile)
	err := os.MkdirAll("media"+path, 0777)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("media"+path+"/"+segmentFile.name, data, 0644); err != nil {
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

func main() {
	extractTitleAudio()
	//fmt.Println(compileSegmentFileInfo())
	//fmt.Println(compileSegmentDirInfo())
}

func decompressChunks() {
	// decompression of chunks
	chunkInfo, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/CONTENT.tda.tdz")
	if err != nil {
		panic(err)
	}
	defer chunkInfo.Close()
	chunks, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/CONTENT.tda")
	if err != nil {
		panic(err)
	}
	defer chunks.Close()

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
		zipSize := uint32(binary.LittleEndian.Uint32(b[4:]))
		zip := make([]byte, zipSize)
		if _, err := chunks.Read(zip); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		zipReader, err := zlib.NewReader(bytes.NewReader(zip))
		if err != nil {
			panic(err)
		}
		byteArray := new(bytes.Buffer)
		io.Copy(byteArray, zipReader)
		if err := os.WriteFile("chunks/"+strconv.Itoa(i), byteArray.Bytes(), 0644); err != nil {
			panic(err)
		}
	}
}

func mainaaa() {
	f, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/CONTENT.tda.tdz")
	if err != nil {
		log.Fatal("fail")
	}
	defer f.Close()

	// c, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/CONTENT.tda")
	// if err != nil {
	// 	log.Fatal("fail")
	// }
	// defer c.Close()

	i := 0
	b := make([]byte, 4)
	for ; ; i++ {
		n, err := f.Read(b)
		if err != nil {
			break
			//log.Fatal("fail", err)
		}
		if n != 4 {
			break
		}
	}

	fmt.Println(i)
}

func maine() {
	dat, err := os.ReadFile("/home/ryoji/archive/ldoce5/ldoce5.data/gb_hwd_pron.skn/files.skn/NAME.tda")
	if err != nil {
		log.Fatal("fail")
	}
	//47350
	names := bytes.Split(dat, []byte("\000"))
	//for _, n := range names {
	//	fmt.Println(string(n))
	//}
	f, err := os.Open("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/files.skn/files.dat")
	if err != nil {
		log.Fatal("fail")
	}
	defer f.Close()

	i := 0
	b := make([]byte, 16)
	for ; ; i++ {
		n, err := f.Read(b)
		if err != nil {
			break
			//log.Fatal("fail", err)
		}
		if n != 16 {
			break
		}
		// fmt.Println(uint32(binary.LittleEndian.Uint32(b)))
	}
	// 47350
	fmt.Println(i, len(names))
}

func maind() {
	f, err := os.Open("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/dirs.skn/DIRS.skn/DIRS.dat")
	if err != nil {
		log.Fatal("fail")
	}
	defer f.Close()

	i := 0
	b := make([]byte, 2)
	for ; ; i++ {
		n, err := f.Read(b)
		if err != nil {
			break
			//log.Fatal("fail", err)
		}
		if n != 2 {
			break
		}
		// fmt.Println(uint32(binary.LittleEndian.Uint32(b)))
	}
	// 21704
	fmt.Println(i)
}

func mainc() {
	f, err := os.Open("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/dirs.skn/FILES.skn/FILES.dat")
	if err != nil {
		log.Fatal("fail")
	}
	defer f.Close()

	i := 0
	b := make([]byte, 2)
	for ; ; i++ {
		n, err := f.Read(b)
		if err != nil {
			break
			//log.Fatal("fail", err)
		}
		if n != 2 {
			break
		}
		// fmt.Println(uint32(binary.LittleEndian.Uint32(b)))
	}
	// 47350
	fmt.Println(i)
}

func mainaa() {
	dat, err := os.ReadFile("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/dirs.skn/NAME.tda")
	if err != nil {
		log.Fatal("fail")
	}
	// 21706
	names := bytes.Split(dat, []byte("\000"))
	// for _, n := range names {
	// 	fmt.Println(string(n))
	// }
	fmt.Println(len(names))
	f, err := os.Open("/home/ryoji/archive/ldoce5.data/gb_hwd_pron.skn/dirs.skn/dirs.dat")
	if err != nil {
		log.Fatal("fail")
	}
	defer f.Close()

	i := 0
	b := make([]byte, 9)
	for ; ; i++ {
		n, err := f.Read(b)
		if err != nil {
			break
			//log.Fatal("fail", err)
		}
		if n != 9 {
			break
		}
		// fmt.Println(uint32(binary.LittleEndian.Uint32(b)))
	}
	// 21706
	fmt.Println(i, len(names))
}

func maina() {
	dat, err := os.ReadFile("/home/ryoji/archive/ldoce5/ldoce5.data/exa_pron.skn/files.skn/NAME.tda")
	if err != nil {
		log.Fatal("fail")
	}
	names := bytes.Split(dat, []byte("\000"))
	for _, n := range names {
		fmt.Println(string(n))
	}
	fmt.Println(len(names))
	f, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/exa_pron.skn/files.skn/CONTENT.tda.tdz")
	if err != nil {
		log.Fatal("fail")
	}
	defer f.Close()

	c, err := os.Open("/home/ryoji/archive/ldoce5/ldoce5.data/exa_pron.skn/files.skn/CONTENT.tda")
	if err != nil {
		log.Fatal("fail")
	}
	defer c.Close()

	i := 0
	b := make([]byte, 8)
	for ; ; i++ {
		n, err := f.Read(b)
		if err != nil {
			log.Fatal("fail", err)
		}
		if n != 8 {
			break
		}
		l := binary.LittleEndian.Uint32(b[4:])
		size := uint32(l)
		fmt.Println(size)
		z := make([]byte, size)
		a, err := c.Read(z)
		if err != nil {
			log.Fatal("fail", err)
		}
		if a < 1 {
			break
		}
		if err := os.WriteFile(string(names[i]), z, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println(uint32(binary.LittleEndian.Uint32(b)))
	}

}
