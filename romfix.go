package main

import (
	"archive/zip"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

type ROMFile struct {
	Name string
	SourceZip string
	SourceName string
}

type ZipToWrite struct {
	Name string
	ROMs []ROMFile
}

type GameZip struct {
	Name string
	Err  error
	ROMs []ROM
}

type ROM struct {
	Name string
	Size uint64
	CRC  uint32
	SHA1 []byte
}

type MameXML struct {
	Build string    `xml:"build,attr"`
	Games []GameXML `xml:"game"`
}

type GameXML struct {
	Name        string   `xml:"name,attr"`
	Parent      string   `xml:"cloneof,attr"`
	BIOS        string   `xml:"romof,attr"`
	SourceFile  string   `xml:"sourcefile,attr"`
	Description string   `xml:"description"`
	ROMs        []ROMXML `xml:"rom"`
}

type ROMXML struct {
	Name   string `xml:"name,attr"`
	Size   uint64 `xml:"size,attr"`
	CRCS   string `xml:"crc,attr"`
	SHA1S  string `xml:"sha1,attr"`
	Status string `xml:"status,attr"`
	CRC    uint32
	SHA1   []byte
}

func readZips() []GameZip {
	romPath := "/home/jeffm/jeff/mame/roms-0153"

	zipFiles, err := ioutil.ReadDir(romPath)
	if err != nil {
		fmt.Printf("Could not open rom dir: %v\n", err)
		return nil
	}

	gameZips := make([]GameZip, 0, 32)

	for _, zipFile := range zipFiles {
		splitName := strings.Split(zipFile.Name(), ".")
		if len(splitName) != 2 || splitName[1] != "zip" {
			// not a rom
			continue
		}
		gameZip := GameZip{Name: splitName[0]}

		zipPath := filepath.Join(romPath, zipFile.Name())
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			gameZip.Err = fmt.Errorf("Error opening rom %s: %v",
				zipFile.Name(), err)
			continue
		}
		defer zipReader.Close()

		for _, romFile := range zipReader.File {
			// romReader, err := romFile.Open()
			// if err != nil {
			// 	continue
			// }
			// defer romReader.Close()
			// hsh := sha1.New()
			// io.Copy(hsh, romReader)
			// sha := hsh.Sum(nil)
			// rom := ROM{Name: romFile.Name, SHA1: sha,
			// 	CRC: romFile.CRC32}
			rom := ROM{Name: romFile.Name,
				CRC:  romFile.CRC32,
				Size: romFile.UncompressedSize64}
			gameZip.ROMs = append(gameZip.ROMs, rom)
		}
		gameZips = append(gameZips, gameZip)
	}
	return gameZips
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
		if game.Parent != "" {
			for _, info := range mame.Games {
				if info.Name == game.Parent {
					mame.Games[gameI].BIOS = info.BIOS
				}
			}
		}
		roms := make([]ROMXML, 0, len(game.ROMs))
		for _, rom := range game.ROMs {
			if rom.Status != "nodump" {
				new_rom := ROMXML{Name: rom.Name,
					Size: rom.Size}
				crc, err := strconv.ParseUint(rom.CRCS, 16, 32)
				if err != nil {
					return nil, fmt.Errorf(
						"Error converting rom crc %s %s: %v",
						game.Name, rom.Name, err)
				}
				new_rom.CRC = uint32(crc)
				new_rom.SHA1, err = hex.DecodeString(rom.SHA1S)
				if err != nil {
					return nil, fmt.Errorf(
						"Error converting rom sha1: %v", err)
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

// func findROMfile(name string, files []*zip.File) (*zip.File, error) {
// 	for _, file := range files {
// 		if file.Name == name {
// 			return file, nil
// 		}
// 	}
// 	return nil, fmt.Errorf("ROM %s not found in zip", name)
// }

func findROM(name string, gameZip GameZip, parentZip, biosZip *GameZip) (*ROM, error) {
	for _, rom := range gameZip.ROMs {
		if rom.Name == name {
			return &rom, nil
		}
	}
	if parentZip != nil {
		for _, rom := range parentZip.ROMs {
			if rom.Name == name {
				return &rom, nil
			}
		}
	}
	if biosZip != nil {
		for _, rom := range biosZip.ROMs {
			if rom.Name == name {
				return &rom, nil
			}
		}
	}
	return nil, fmt.Errorf("ROM %s not found in zip", name)
}

// func validate(zipFile os.FileInfo, mame *MameXML) []error {
// 	errors := make([]error, 0, 10)
// 	romPath := "/home/jeffm/jeff/mame/roms-0153"

// 	splitName := strings.Split(zipFile.Name(), ".")
// 	if len(splitName) != 2 || splitName[1] != "zip" {
// 		return append(errors, fmt.Errorf("%v not a rom", zipFile.Name()))
// 	}
// 	game, err := findGame(splitName[0], mame)
// 	if err != nil {
// 		return append(errors, err)
// 	}

// 	romFileName := filepath.Join(romPath, zipFile.Name())
// 	romZip, err := zip.OpenReader(romFileName)
// 	if err != nil {
// 		return append(errors, fmt.Errorf("Error opening rom %s: %v",
// 			romFileName, err))
// 	}
// 	defer romZip.Close()

// 	for _, romInfo := range game.ROMs {
// 		rom, err := findROMfile(romInfo.Name, romZip.File)
// 		if err != nil {
// 			errors = append(errors, fmt.Errorf("game %s: %v",
// 				game.Name, err))
// 			continue
// 		}
// 		romReader, err := rom.Open()
// 		if err != nil {
// 			errors = append(errors, fmt.Errorf(
// 				"error opening rom inside zip %s: %v",
// 				rom.Name, err))
// 			continue
// 		}
// 		defer romReader.Close()
// 		hsh := sha1.New()
// 		io.Copy(hsh, romReader)
// 		sha := hsh.Sum(nil)
// 		if !bytes.Equal(sha, romInfo.SHA1) {
// 			errors = append(errors, fmt.Errorf(
// 				"game %s: Invalid rom %s, found %x, expected %x",
// 				game.Name, rom.Name, sha, romInfo.SHA1))
// 			continue
// 		}
// 	}
// 	return errors
// }

func findGameZip(name string, gameZips []GameZip) (*GameZip, error) {
	for _, zip := range gameZips {
		if zip.Name == name {
			return &zip, nil
		}
	}
	return nil, fmt.Errorf("GameZip %s not found in romdir", name)
}

func fixROM(missing ROMXML, gameZips []GameZip, gameInfo *GameXML, mameInfo *MameXML) {
	//Check if missing rom belongs to parent or bios
	if gameInfo.BIOS != "" {
		for _, info := range mameInfo.Games {
			if info.Name == gameInfo.BIOS {
				for _, romInfo := range info.ROMs {
					if romInfo.Name == missing.Name {
						fmt.Printf("Missing rom should be in bios %s\n", info.Name)
						return
					}
				}
			}
		}
	}
	if gameInfo.Parent != "" {
		for _, info := range mameInfo.Games {
			if info.Name == gameInfo.Parent {
				for _, romInfo := range info.ROMs {
					if romInfo.Name == missing.Name {
						fmt.Printf("Missing rom should be in parent %s\n", info.Name)
						return
					}
				}
			}
		}
	}
	for _, gameZip := range gameZips {
		for _, rom := range gameZip.ROMs {
			if rom.Size == missing.Size && rom.CRC == missing.CRC {
				fmt.Printf("Missing ROM found\n")
				fmt.Printf("Looking for %v %v 0x%x\n", missing.Name, missing.Size, missing.CRC)
				fmt.Printf("found in %v %v\n", gameZip.Name, rom.Name)
				fmt.Printf("size crc %v 0x%x\n", rom.Size, rom.CRC)
			}
		}
	}
}

func findProblems(mameInfo *MameXML, gameZips []GameZip) {
	for _, gameZip := range gameZips {
		gameInfo, err := findGame(gameZip.Name, mameInfo)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			continue
		}
		var parentZip *GameZip
		var biosZip *GameZip
		if gameInfo.Parent != "" {
			parentZip, err = findGameZip(gameInfo.Parent, gameZips)
			if err != nil {
				fmt.Printf("game %s: %v\n", gameInfo.Name, err)
				continue
			}
		}
		if gameInfo.BIOS != "" {
			biosZip, err = findGameZip(gameInfo.BIOS, gameZips)
			if err != nil {
				fmt.Printf("game %s: %v\n", gameInfo.Name, err)
				continue
			}
		}
		for _, romInfo := range gameInfo.ROMs {
			rom, err := findROM(romInfo.Name, gameZip, parentZip, biosZip)
			if err != nil {
				fmt.Printf("game %s: %v\n", gameInfo.Name, err)
				fixROM(romInfo, gameZips, gameInfo, mameInfo)
				continue
			}
			if rom.Size != romInfo.Size {
				fmt.Printf("game %s: rom %s: size invalid\n", gameInfo.Name, romInfo.Name)
				fixROM(romInfo, gameZips, gameInfo, mameInfo)
				continue
			}
			if rom.CRC != romInfo.CRC {
				fmt.Printf("game %s: rom %s: crc invalid\n", gameInfo.Name, romInfo.Name)
				fixROM(romInfo, gameZips, gameInfo, mameInfo)
				continue
			}
		}
	}
}

func main() {
	mame, err := readXML()

	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	// printDebug(mame)

	gameZips := readZips()

	findProblems(mame, gameZips)

	// romPath := "/home/jeffm/jeff/mame/roms-0153"

	// zipFiles, err := ioutil.ReadDir(romPath)
	// if err != nil {
	// 	fmt.Printf("error: %v\n", err)
	// 	return
	// }

	// for _, zipFile := range zipFiles {
	// 	errs := validate(zipFile, mame)
	// 	if len(errs) != 0 {
	// 		for _, err := range errs {
	// 			fmt.Printf("invalid: %v\n", err)
	// 		}
	// 	} else {
	// 		//fmt.Printf("%s is valid\n", zipFile.Name())
	// 	}
	// }
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
