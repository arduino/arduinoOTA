package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const AppVersion = "1.1.1"

var compileInfo string

var (
	version        = flag.Bool("version", false, "Prints program version")
	networkAddress = flag.String("address", "localhost", "The address of the board")
	networkPort    = flag.String("port", "80", "The board needs to be listening on this port")
	sketchPath     = flag.String("sketch", "", "Sketch path")
	uploadEndpoint = flag.String("upload", "", "Upload endpoint")
	resetEndpoint  = flag.String("reset", "", "Upload endpoint")
	syncEndpoint   = flag.String("sync", "", "Upload endpoint")
	binMode        = flag.Bool("b", false, "Upload binary mode")
	verbose        = flag.Bool("v", true, "Verbose flag")
	quiet          = flag.Bool("q", false, "Quiet flag")
	useSsl         = flag.String("ssl", "", "SSL flag")
	syncRet        = flag.String("sync_exp", "", "sync expected return code in format code:string")
)

type Item struct {
	Id   int
	Name string
}

func main() {
	flag.Parse()

	if *version {
		fmt.Println(AppVersion + compileInfo)
		os.Exit(0)
	}

	httpheader := "http://"

	if *useSsl != "" {
		httpheader = "https://"
	}

	syncRetCode := 200
	syncString := "SYNC"

	if *syncRet != "" {
		sliceStrRet := strings.Split(*syncRet, ":")
		if len(sliceStrRet) == 2 {
			syncRetCode, _ = strconv.Atoi(sliceStrRet[0])
			syncString = sliceStrRet[1]
		}
	}

	if *syncEndpoint != "" {
		if *verbose {
			fmt.Println("Resetting the board")
		}

		resp, err := http.Post(httpheader+*networkAddress+":"+*networkPort+*syncEndpoint, "", nil)
		if err != nil || resp.StatusCode != syncRetCode {
			if *verbose {
				fmt.Println("Failed to reset the board, upload failed")
			}
			os.Exit(1)
		}
		defer resp.Body.Close()
	}

	if *syncEndpoint != "" {
		if *verbose {
			fmt.Println("Waiting for the upload to start")
		}

		timeout := 0

		for timeout < 10 {
			resp, err := http.Get(httpheader + *networkAddress + ":" + *networkPort + *syncEndpoint)
			if err != nil {
				if *verbose {
					fmt.Println("Failed to reset the board, upload failed")
				}
				os.Exit(1)
			}
			defer resp.Body.Close()

			statusString, err := ioutil.ReadAll(resp.Body)

			if strings.Contains(string(statusString), syncString) {
				fmt.Println(string(statusString))
				break
			}

			time.Sleep(1 * time.Second)
			timeout++
		}
	}

	if *uploadEndpoint != "" {
		if *verbose {
			fmt.Println("Uploading the sketch")
		}

		f, err := os.Open(*sketchPath)
		if err != nil {
			if *verbose {
				fmt.Println("Failed to open the sketch")
			}
			os.Exit(1)
		}
		defer f.Close()

		var sketchData *bytes.Buffer

		if *binMode {
			sketchData = StreamToBytes(f)
		} else {
			str := StreamToString(f)
			re := regexp.MustCompile(`\r?\n`)
			str = re.ReplaceAllString(str, "")
			sketchData = bytes.NewBufferString(str)
		}

		req, err := http.NewRequest("POST", httpheader+*networkAddress+":"+*networkPort+*uploadEndpoint, sketchData)
		if err != nil {
			if *verbose {
				fmt.Println("Error sending sketch file")
			}
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if *verbose {
				fmt.Println("Error flashing the sketch")
			}
			os.Exit(1)
		}
		defer resp.Body.Close()

		respStr, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != 200 {
			if *verbose {
				fmt.Println("Error flashing the sketch:" + string(respStr))
			}
			os.Exit(1)
		}

		if *verbose {
			fmt.Println(string(respStr))
			fmt.Println("Sketch uploaded successfully")
		}
	}

	if *resetEndpoint != "" {
		if *verbose {
			fmt.Println("Resetting the board")
		}

		resp, err := http.Post(httpheader+*networkAddress+":"+*networkPort+*resetEndpoint, "", nil)
		if err != nil {
			if *verbose {
				fmt.Println("Failed to reset the board, please reset maually")
			}
			os.Exit(0)
		}
		defer resp.Body.Close()
	}
}

func StreamToBytes(stream io.Reader) *bytes.Buffer {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf
}

func StreamToString(stream io.Reader) string {
	return StreamToBytes(stream).String()
}
