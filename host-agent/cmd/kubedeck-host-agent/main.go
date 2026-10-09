package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var portPattern = regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+) -> (\d+)`)

type secret struct {
	Data map[string]string `json:"data"`
}

type apiResult struct {
	Status       string          `json:"status"`
	ErrorMessage string          `json:"errorMessage"`
	Token        string          `json:"token"`
	Response     json.RawMessage `json:"response"`
}

type apiClient struct {
	base  string
	token string
	http  *http.Client
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "host-agent configuration failed:", err)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := reconcile(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "host-agent DNS reconciliation failed:", err)
		os.Exit(1)
	}
}

func reconcile(ctx context.Context, cfg config) error {
	var ip net.IP
	if cfg.TargetIP != "" {
		ip = net.ParseIP(cfg.TargetIP).To4()
	} else {
		var err error
		ip, err = lanIP(ctx, cfg.InterfaceName)
		if err != nil {
			return err
		}
	}
	password, err := adminPassword(ctx, cfg)
	if err != nil {
		return err
	}
	port, stop, err := forwardAPI(ctx, cfg)
	if err != nil {
		return err
	}
	defer stop()
	client := apiClient{base: "http://127.0.0.1:" + port, http: &http.Client{Timeout: 8 * time.Second}}
	login, err := client.call(ctx, "/api/user/login", url.Values{"user": {cfg.AdminUser}, "pass": {password}})
	if err != nil {
		return fmt.Errorf("Technitium login: %w", err)
	}
	client.token = login.Token
	if client.token == "" {
		return errors.New("Technitium login returned no token")
	}
	defer func() { _, _ = client.call(context.Background(), "/api/user/logout", url.Values{}) }()
	return reconcileDNS(ctx, &client, cfg, ip)
}

func reconcileDNS(ctx context.Context, client *apiClient, cfg config, ip net.IP) error {
	zones, err := client.call(ctx, "/api/zones/list", nil)
	if err != nil {
		return err
	}
	var zoneList struct {
		Zones []struct {
			Name string `json:"name"`
		} `json:"zones"`
	}
	if err := json.Unmarshal(zones.Response, &zoneList); err != nil {
		return err
	}
	found := false
	for _, z := range zoneList.Zones {
		if strings.EqualFold(z.Name, cfg.Zone) {
			found = true
			break
		}
	}
	if !found {
		if _, err := client.call(ctx, "/api/zones/create", url.Values{"zone": {cfg.Zone}, "type": {"Primary"}}); err != nil {
			return err
		}
	}

	records, err := client.call(ctx, "/api/zones/records/get?"+url.Values{"domain": {cfg.recordName()}, "zone": {cfg.Zone}}.Encode(), nil)
	if err != nil {
		return err
	}
	var recordList struct {
		Records []struct {
			Name  string `json:"name"`
			Type  string `json:"type"`
			TTL   int    `json:"ttl"`
			RData struct {
				IPAddress string `json:"ipAddress"`
			} `json:"rData"`
		} `json:"records"`
	}
	if err := json.Unmarshal(records.Response, &recordList); err != nil {
		return err
	}
	for _, r := range recordList.Records {
		if strings.EqualFold(r.Name, cfg.recordName()) && r.Type == "A" && r.TTL == 30 && r.RData.IPAddress == ip.String() {
			fmt.Println("Technitium wildcard current:", ip)
			return nil
		}
	}
	_, err = client.call(ctx, "/api/zones/records/add", url.Values{
		"domain": {cfg.recordName()}, "zone": {cfg.Zone}, "type": {"A"}, "ttl": {"30"}, "overwrite": {"true"}, "ipAddress": {ip.String()},
	})
	if err != nil {
		return err
	}
	fmt.Println("Technitium wildcard updated:", ip)
	return nil
}

func lanIP(ctx context.Context, override string) (net.IP, error) {
	device := override
	if device == "" {
		out, err := exec.CommandContext(ctx, "route", "-n", "get", "default").Output()
		if err != nil {
			return nil, fmt.Errorf("find default route: %w", err)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "interface:" {
				device = fields[1]
				break
			}
		}
	}
	if device == "" || strings.HasPrefix(device, "utun") {
		return nil, fmt.Errorf("no physical LAN interface from default route (%s); use -interface", device)
	}
	ifc, err := net.InterfaceByName(device)
	if err != nil {
		return nil, err
	}
	if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
		return nil, fmt.Errorf("interface %s is not an active LAN interface", device)
	}
	addrs, err := ifc.Addrs()
	if err != nil {
		return nil, err
	}
	for _, a := range addrs {
		ip, _, err := net.ParseCIDR(a.String())
		if err == nil && ip.To4() != nil && ip.IsPrivate() && !ip.IsLinkLocalUnicast() {
			return ip.To4(), nil
		}
	}
	return nil, fmt.Errorf("interface %s has no private IPv4 address", device)
}

func adminPassword(ctx context.Context, cfg config) (string, error) {
	out, err := exec.CommandContext(ctx, "kubectl", cfg.kubectlArgs("get", "secret", cfg.AdminSecret, "-o", "json")...).Output()
	if err != nil {
		return "", fmt.Errorf("read Technitium Kubernetes credential: %w", err)
	}
	var s secret
	if err := json.Unmarshal(out, &s); err != nil {
		return "", err
	}
	encoded := s.Data[cfg.PasswordKey]
	if encoded == "" {
		return "", errors.New("Technitium credential key is missing")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func forwardAPI(parent context.Context, cfg config) (string, func(), error) {
	ctx, cancel := context.WithCancel(parent)
	cmd := exec.CommandContext(ctx, "kubectl", cfg.kubectlArgs("port-forward", "--address", "127.0.0.1", "svc/"+cfg.Service, ":"+strconv.Itoa(cfg.APIPort))...)
	reader, writer := io.Pipe()
	cmd.Stdout, cmd.Stderr = writer, writer
	if err := cmd.Start(); err != nil {
		cancel()
		return "", nil, err
	}
	lines := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			default:
				// Port-forward emits connection lines after startup; never block its pipe.
			case <-ctx.Done():
				return
			}
		}
		close(lines)
	}()
	stop := func() { cancel(); _ = cmd.Wait(); _ = writer.Close(); _ = reader.Close() }
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				stop()
				return "", nil, errors.New("kubectl port-forward exited")
			}
			if matches := portPattern.FindStringSubmatch(line); len(matches) == 3 && matches[2] == strconv.Itoa(cfg.APIPort) {
				return matches[1], stop, nil
			}
		case <-timer.C:
			stop()
			return "", nil, errors.New("timed out starting Technitium port-forward")
		case <-parent.Done():
			stop()
			return "", nil, parent.Err()
		}
	}
}

func (c *apiClient) call(ctx context.Context, path string, form url.Values) (apiResult, error) {
	var body io.Reader
	method := http.MethodGet
	if form != nil {
		method, body = http.MethodPost, bytes.NewBufferString(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return apiResult{}, err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return apiResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return apiResult{}, fmt.Errorf("Technitium %s: HTTP %d", path, resp.StatusCode)
	}
	var result apiResult
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return apiResult{}, err
	}
	if result.Status != "ok" {
		return apiResult{}, fmt.Errorf("Technitium %s: %s", path, result.ErrorMessage)
	}
	return result, nil
}
