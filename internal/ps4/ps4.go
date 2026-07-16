package ps4

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
)

const ftpPort = "2121"

var titleIDRe = regexp.MustCompile(`[a-zA-Z0-9]{4}[0-9]{5}`)

var ps4Prefixes = map[string]bool{"CUSA": true}

var classicPrefixes = map[string]bool{
	"SLES": true, "SCES": true, "SCED": true, "SLUS": true, "SCUS": true,
	"SLPS": true, "SCAJ": true, "SLKA": true, "SLPM": true, "SCPS": true,
	"CF00": true, "SCKA": true, "ALCH": true, "CPCS": true, "SLAJ": true,
	"KOEI": true, "ARZE": true, "TCPS": true, "SCCS": true, "PAPX": true,
	"SRPM": true, "GUST": true, "WLFD": true, "ULKS": true, "VUGJ": true,
	"HAKU": true, "ROSE": true, "CZP2": true, "ARP2": true, "PKP2": true,
	"SLPN": true, "NMP2": true, "MTP2": true, "SCPM": true, "PBPX": true,
}

func dial(ip string) (*ftp.ServerConn, error) {
	return ftp.Dial(net.JoinHostPort(ip, ftpPort),
		ftp.DialWithTimeout(5*time.Second),
		ftp.DialWithDisabledEPSV(true))
}

func TestForPS4(ip string) bool {
	c, err := dial(ip)
	if err != nil {
		fmt.Printf("TestForPS4():     No FTP server found on '%s'. '%v'.\n", ip, err)
		return false
	}
	defer c.Quit()
	if err := c.Login("anonymous", "anonymous"); err != nil {
		fmt.Printf("TestForPS4():     login failed on '%s'. '%v'.\n", ip, err)
		return false
	}
	entries, err := c.NameList("/mnt/sandbox")
	if err != nil {
		fmt.Printf("TestForPS4():     No /mnt/sandbox on '%s'. '%v'.\n", ip, err)
		return false
	}
	for _, e := range entries {
		base := e[strings.LastIndex(e, "/")+1:]
		if strings.HasPrefix(base, "NPXS20001_") {
			fmt.Printf("TestForPS4():     PS4 found on '%s'\n", ip)
			return true
		}
	}
	fmt.Printf("TestForPS4():     NPXS20001 (shell UI) sandbox not found on '%s'.\n", ip)
	return false
}

func GetTitleID(ip string) (titleID, gameType string, ok bool) {
	c, err := dial(ip)
	if err != nil {
		return "", "", false
	}
	defer c.Quit()
	if err := c.Login("anonymous", "anonymous"); err != nil {
		return "", "", false
	}
	entries, err := c.NameList("/mnt/sandbox")
	if err != nil {
		return "", "", false
	}

	// RE2 has no lookahead, so skip NPXS entries before matching the titleID.
	for _, e := range entries {
		base := e[strings.LastIndex(e, "/")+1:]
		if strings.HasPrefix(base, "NPXS") {
			continue
		}
		if m := titleIDRe.FindString(base); m != "" {
			titleID = m
		}
	}

	if titleID == "" {
		return "main_menu", "", true
	}
	prefix := titleID[:4]
	switch {
	case ps4Prefixes[prefix]:
		gameType = "PS4"
	case classicPrefixes[prefix]:
		gameType = "PS1/2"
	default:
		gameType = "Homebrew"
	}
	return titleID, gameType, true
}

func PromptUser() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Get PS4's IP address Automatically or Manually?")
	for {
		fmt.Print("Please enter either 'a' or 'm': ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" {
			continue
		}
		switch line[0] {
		case 'a':
			if ip := ScanNetwork(); ip != "" {
				return ip
			}
			fmt.Println("No device on network was found to belong to a Jailbroken PS4 running an FTP server.")
		case 'm':
			return GetIPFromUser()
		}
	}
}

func GetIPFromUser() string {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Please enter the PS4's IP address: ")
		line, _ := reader.ReadString('\n')
		ip := strings.TrimSpace(line)
		if ip != "" && TestForPS4(ip) {
			return ip
		}
	}
}

func ScanNetwork() string {
	host, err := hostIP()
	if err != nil {
		fmt.Printf("Error while getting host network. '%v'\n", err)
		return GetIPFromUser()
	}
	parts := strings.Split(host, ".")
	prefix := strings.Join(parts[:3], ".") + "."
	fmt.Printf("Expected network is '%s0/24'. Scanning...\n", prefix)

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		found string
		sem   = make(chan struct{}, 32)
	)
	for i := 1; i < 255; i++ {
		ip := fmt.Sprintf("%s%d", prefix, i)
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			mu.Lock()
			done := found != ""
			mu.Unlock()
			if done {
				return
			}
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, ftpPort), 2*time.Second)
			if err != nil {
				return
			}
			conn.Close()
			if TestForPS4(ip) {
				mu.Lock()
				if found == "" {
					found = ip
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return found
}

func hostIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}
