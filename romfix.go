package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type MameXML struct {
	Build string    `xml:"build,attr"`
	Games []GameXML `xml:"game"`
}

type GameXML struct {
	Name        string   `xml:"name,attr"`
	SourceFile  string   `xml:"sourcefile,attr"`
	Description string   `xml:"description"`
	ROMs        []ROMXML `xml:"rom"`
}

type ROMXML struct {
	Name   string `xml:"name,attr"`
	Size   int    `xml:"size,attr"`
	CRCS   string `xml:"crc,attr"`
	SHA1S  string `xml:"sha1,attr"`
	Status string `xml:"status,attr"`
	CRC    uint32
	SHA1   []byte
}

func readXML() (*MameXML, error) {
	mameXml, err := ioutil.ReadFile("/home/jeffm/jeff/mame/allmame.xml")
	if err != nil {
		return nil, fmt.Errorf("Error reading mame xml: %v", err)
	}

	var mame MameXML
	err = xml.Unmarshal(mameXml, &mame)
	if err != nil {
		return nil, fmt.Errorf("Error parsing mame xml: %v", err)
	}

	for gameI, game := range mame.Games {
		roms := make([]ROMXML, 0, len(game.ROMs))
		for _, rom := range game.ROMs {
			if rom.Status != "nodump" {
				new_rom := ROMXML{Name: rom.Name, Size: rom.Size}
				crc, err := strconv.ParseUint(rom.CRCS, 16, 32)
				if err != nil {
					return nil, fmt.Errorf("Error converting rom crc %s %s: %v", game.Name, rom.Name, err)
				}
				new_rom.CRC = uint32(crc)
				new_rom.SHA1, err = hex.DecodeString(rom.SHA1S)
				if err != nil {
					return nil, fmt.Errorf("Error converting rom sha1: %v", err)
				}
				roms = append(roms, new_rom)
			}
		}
		mame.Games[gameI].ROMs = roms
	}
	return &mame, nil
}

// func printDebug(mame *MameXML) {
// 	fmt.Printf("Build: %v\n", mame.Build)
// 	for _, game := range mame.Games {
// 		fmt.Printf("Name: %v\n", game.Name)
// 		for _, rom := range game.ROMs {
// 			fmt.Printf("Name:    %v\n", rom.Name)
// 			fmt.Printf("Size:    %v\n", rom.Size)
// 			fmt.Printf("CRC:     0x%x\n", rom.CRC)
// 			fmt.Printf("SHA1:    %x\n", rom.SHA1)
// 		}
// 	}
// }

func findGame(name string, mame *MameXML) (*GameXML, error) {
	for _, game := range mame.Games {
		if game.Name == name {
			return &game, nil
		}
	}
	return nil, fmt.Errorf("Game %s not found in xml", name)
}

func findROM(name string, files []*zip.File) (*zip.File, error) {
	for _, file := range files {
		if file.Name == name {
			return file, nil
		}
	}
	return nil, fmt.Errorf("ROM %s not found in zip", name)
}

func validate(myROM os.FileInfo, mame *MameXML) []error {
	errors := make([]error, 0, 10)
	romPath := "/home/jeffm/jeff/mame/roms-0153"

	splitName := strings.Split(myROM.Name(), ".")
	if len(splitName) != 2 || splitName[1] != "zip" {
		return append(errors, fmt.Errorf("%v not a rom", myROM.Name()))
	}
	game, err := findGame(splitName[0], mame)
	if err != nil {
		return append(errors, err)
	}

	romFileName := filepath.Join(romPath, myROM.Name())
	romZip, err := zip.OpenReader(romFileName)
	if err != nil {
		return append(errors, fmt.Errorf("Error opening rom %s: %v", romFileName, err))
	}
	defer romZip.Close()

	for _, romInfo := range game.ROMs {
		rom, err := findROM(romInfo.Name, romZip.File)
		if err != nil {
			errors = append(errors, fmt.Errorf("game %s: %v", game.Name, err))
			continue
		}
		romReader, err := rom.Open()
		if err != nil {
			errors = append(errors, fmt.Errorf("error opening rom inside zip %s: %v", rom.Name, err))
			continue
		}
		defer romReader.Close()
		hsh := sha1.New()
		io.Copy(hsh, romReader)
		sha := hsh.Sum(nil)
		if !bytes.Equal(sha, romInfo.SHA1) {
			errors = append(errors, fmt.Errorf("game %s: Invalid rom %s, found %x, expected %x", game.Name, rom.Name, sha, romInfo.SHA1))
			continue
		}
	}
	return errors
}

func main() {
	mame, err := readXML()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	// printDebug(mame)

	romPath := "/home/jeffm/jeff/mame/roms-0153"

	myROMs, err := ioutil.ReadDir(romPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	for _, myROM := range myROMs {
		errs := validate(myROM, mame)
		if len(errs) != 0 {
			for _, err := range errs {
				fmt.Printf("invalid: %v\n", err)
			}
		} else {
			//fmt.Printf("%s is valid\n", myROM.Name())
		}
	}
}

// func blockHash(romReader io.Reader) {
// 	hsh := sha1.New()
// 	// bs := hsh.BlockSize()
// 	t0 := time.Now()
// 	// var err error
// 	// for err == nil {
// 	// 	_, err = io.CopyN(hsh, romReader, int64(bs))
// 	// }
// 	io.Copy(hsh, romReader)
// 	sha := hsh.Sum(nil)
// 	t1 := time.Now()
// 	fmt.Printf("The block hash took %v to run.\n", t1.Sub(t0))
// 	fmt.Printf("SHA1: 0x%x\n", sha)
// }

// func readAndHash(romReader io.Reader) {
// 	t0 := time.Now()
// 	contents, err := ioutil.ReadAll(romReader)
// 	t1 := time.Now()
// 	fmt.Printf("The read took %v to run.\n", t1.Sub(t0))
// 	if err != nil {
// 		fmt.Printf("error: %v", err)
// 		return
// 	}
// 	t0 = time.Now()
// 	sha := sha1.Sum(contents)
// 	t1 = time.Now()
// 	fmt.Printf("SHA1: 0x%x\n", sha)
// 	fmt.Printf("The SHA1 took %v to run.\n", t1.Sub(t0))
// 	// t0 = time.Now()
// 	// crc := crc32.ChecksumIEEE(contents)
// 	// t1 = time.Now()
// 	// fmt.Printf("CRC: 0x%x\n", crc)
// 	// fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
// }
