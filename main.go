package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/arduino/arduinoOTA/globals"
)

var compileInfo string

var (
	version         = flag.Bool("version", false, "Prints program version")
	networkAddress  = flag.String("address", "localhost", "The address of the board")
	networkPort     = flag.String("port", "80", "The board needs to be listening on this port")
	username        = flag.String("username", "", "Username for authentication")
	password        = flag.String("password", "", "Password for authentication")
	sketchPath      = flag.String("sketch", "", "Sketch path")
	uploadEndpoint  = flag.String("upload", "", "Upload endpoint")
	resetEndpoint   = flag.String("reset", "", "Upload endpoint")
	syncEndpoint    = flag.String("sync", "", "Upload endpoint")
	binMode         = flag.Bool("b", false, "Upload binary mode")
	verbose         = flag.Bool("v", true, "Verbose flag")
	quiet           = flag.Bool("q", false, "Quiet flag")
	useSsl          = flag.String("ssl", "", "SSL flag")
	syncRet         = flag.String("sync_exp", "", "sync expected return code in format code:string")
	hasDownloadFile = flag.Bool("d", false, "set to true to take advantage of downloadFile API")
	timeoutSeconds  = flag.Int("t", 10, "Upload timeout")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println(globals.VersionInfo.String() + compileInfo)
		os.Exit(0)
	}

	var httpClient = &http.Client{
		Timeout: time.Second * time.Duration(*timeoutSeconds),
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

		resp, err := httpClient.Post(httpheader+*networkAddress+":"+*networkPort+*syncEndpoint, "", nil)
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
			resp, err := httpClient.Get(httpheader + *networkAddress + ":" + *networkPort + *syncEndpoint)
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
			sketchData = streamToBytes(f)
		} else {
			str := streamToString(f)
			re := regexp.MustCompile(`\r?\n`)
			str = re.ReplaceAllString(str, "")
			sketchData = bytes.NewBufferString(str)
		}

		if *hasDownloadFile {
			go http.ListenAndServe(":"+*networkPort, http.FileServer(http.Dir(filepath.Dir(*sketchPath))))
			// find my ip if not specified
			ip := getMyIP(net.ParseIP(*networkAddress))
			url := "http://" + ip.String() + ":" + *networkPort + "/" + filepath.Base(*sketchPath)
			sketchData = bytes.NewBufferString(url)
			fmt.Println("Serving sketch on " + url)
		}

		req, err := http.NewRequest("POST", httpheader+*networkAddress+":"+*networkPort+*uploadEndpoint, sketchData)
		if err != nil {
			if *verbose {
				fmt.Println("Error sending sketch file")
			}
			os.Exit(1)
		}

		if *binMode {
			req.Header.Set("Content-Type", "application/octet-stream")
		} else {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		if len(*username) > 0 && len(*password) != 0 {
			req.SetBasicAuth(*username, *password)
		}

		if *verbose {
			trace := &httptrace.ClientTrace{
				ConnectStart: func(network, addr string) {
					fmt.Print("Connecting to board ... ")
				},
				ConnectDone: func(network, addr string, err error) {
					if err != nil {
						fmt.Println("failed!")
					} else {
						fmt.Println(" done")
					}
				},
				WroteHeaders: func() {
					fmt.Print("Uploading sketch ... ")
				},
				WroteRequest: func(wri httptrace.WroteRequestInfo) {
					fmt.Println(" done")
					fmt.Print("Flashing sketch ... ")
				},
				GotFirstResponseByte: func() {
					fmt.Println(" done")
				},
			}
			req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		}

		resp, err := httpClient.Do(req)
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

		resp, err := httpClient.Post(httpheader+*networkAddress+":"+*networkPort+*resetEndpoint, "", nil)
		if err != nil {
			if *verbose {
				fmt.Println("Failed to reset the board, please reset maually")
			}
			os.Exit(0)
		}
		defer resp.Body.Close()
	}
}

func streamToBytes(stream io.Reader) *bytes.Buffer {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf
}

func streamToString(stream io.Reader) string {
	return streamToBytes(stream).String()
}

func getMyIP(otherip net.IP) net.IP {
	ifaces, _ := net.Interfaces()
	// handle err
	var ips []net.IP
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.Contains(otherip) {
					return v.IP
				}
			case *net.IPAddr:
				ips = append(ips, v.IP)
			}
		}
	}
	return nil
}
