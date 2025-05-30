package entity

import (
	"fmt"
	"fwtui/domain/ufw"
	"fwtui/utils/result"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
)

const profilesPath = "/etc/ufw/applications.d/"

type UFWProfile struct {
	Name      string
	Title     string
	Ports     []string
	Installed bool
}

func CreateProfile(p UFWProfile) result.Result[string] {
	// check if file exists
	if _, err := os.Stat(profilesPath + p.Name + ".profile"); !os.IsNotExist(err) {
		return result.Err[string](fmt.Errorf("profile %s already exists", p.Name))
	}

	content := fmt.Sprintf("[%s]\ntitle=%s\ndescription=%s\nports=%s\n",
		p.Name, p.Name, p.Title, strings.Join(p.Ports, "|"))
	err := os.WriteFile(profilesPath+p.Name+".profile", []byte(content), 0644)
	if err != nil {
		return result.Err[string](fmt.Errorf("error creating profile: %s", err))
	}
	ufw.LoadProfile(p.Name)
	return result.Ok(fmt.Sprintf("Profile %s created", p.Name))
}

func DeleteProfile(p UFWProfile) string {
	files, err := os.ReadDir(profilesPath)
	if err != nil {
		return fmt.Sprintf("Error reading profiles directory: %s", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(profilesPath, file.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			continue // optionally log or handle read error
		}

		if strings.Contains(string(content), "["+p.Name+"]") {
			err := os.Remove(path)
			if err != nil {
				return fmt.Sprintf("Error deleting profile: %s", err)
			}
			return fmt.Sprintf("Profile with title '%s' deleted", p.Name)
		}
	}

	return fmt.Sprintf("Profile with title '%s' not found", p.Title)
}

func LoadInstalledProfiles() ([]UFWProfile, error) {

	profileNames := strings.Split(strings.TrimSpace(ufw.GetProfileList()), "\n")[1:]

	var profiles []UFWProfile
	for _, name := range profileNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		profile, err := getUFWProfileInfo(name)
		if err != nil {
			continue
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func getUFWProfileInfo(name string) (UFWProfile, error) {
	lines := strings.Split(ufw.GetProfileInfo(name), "\n")

	profile := UFWProfile{
		Name:      name,
		Installed: true,
	}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Profile:"):
			profile.Name = strings.TrimSpace(strings.TrimPrefix(line, "Profile:"))
		case strings.HasPrefix(line, "Title:"):
			profile.Title = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
		case strings.HasPrefix(line, "Ports:") || strings.HasPrefix(line, "Port:"):
			for j := i + 1; j < len(lines); j++ {
				portLine := strings.TrimSpace(lines[j])
				if portLine == "" {
					break
				}
				profile.Ports = append(profile.Ports, portLine)
			}
		}
	}

	return profile, nil
}

func InstallableProfiles() []UFWProfile {
	installedProfiles, _ := LoadInstalledProfiles()
	installedProfileNames := lo.Map(installedProfiles, func(p UFWProfile, _ int) string {
		return p.Name
	})

	profiles := []UFWProfile{
		// Common access
		{Name: "OpenSSH", Title: "Secure shell access (SSH)", Ports: []string{"22/tcp"}, Installed: lo.Contains(installedProfileNames, "OpenSSH")},
		{Name: "HTTP", Title: "Generic HTTP service", Ports: []string{"80/tcp"}, Installed: lo.Contains(installedProfileNames, "HTTP")},
		{Name: "HTTPS", Title: "Generic HTTPS service", Ports: []string{"443/tcp"}, Installed: lo.Contains(installedProfileNames, "HTTPS")},

		// Web servers
		{Name: "Nginx HTTP", Title: "Nginx web server (HTTP only)", Ports: []string{"80/tcp"}, Installed: lo.Contains(installedProfileNames, "Nginx HTTP")},
		{Name: "Nginx HTTPS", Title: "Nginx web server (HTTPS only)", Ports: []string{"443/tcp"}, Installed: lo.Contains(installedProfileNames, "Nginx HTTPS")},
		{Name: "Nginx Full", Title: "Nginx web server (HTTP and HTTPS)", Ports: []string{"80,443/tcp"}, Installed: lo.Contains(installedProfileNames, "Nginx Full")},
		{Name: "Apache", Title: "Apache web server (HTTP only)", Ports: []string{"80/tcp"}, Installed: lo.Contains(installedProfileNames, "Apache")},
		{Name: "Apache Secure", Title: "Apache web server (HTTPS only)", Ports: []string{"443/tcp"}, Installed: lo.Contains(installedProfileNames, "Apache Secure")},
		{Name: "Apache Full", Title: "Apache web server (HTTP and HTTPS)", Ports: []string{"80,443/tcp"}, Installed: lo.Contains(installedProfileNames, "Apache Full")},

		// Databases
		{Name: "PostgreSQL", Title: "PostgreSQL database server", Ports: []string{"5432/tcp"}, Installed: lo.Contains(installedProfileNames, "PostgreSQL")},
		{Name: "MySQL", Title: "MySQL database server", Ports: []string{"3306/tcp"}, Installed: lo.Contains(installedProfileNames, "MySQL")},
		{Name: "MongoDB", Title: "MongoDB database", Ports: []string{"27017/tcp"}, Installed: lo.Contains(installedProfileNames, "MongoDB")},
		{Name: "Redis", Title: "Redis key-value store", Ports: []string{"6379/tcp"}, Installed: lo.Contains(installedProfileNames, "Redis")},
		{Name: "InfluxDB", Title: "InfluxDB time series database", Ports: []string{"8086/tcp"}, Installed: lo.Contains(installedProfileNames, "InfluxDB")},
		{Name: "Elasticsearch", Title: "Elasticsearch search engine", Ports: []string{"9200,9300/tcp"}, Installed: lo.Contains(installedProfileNames, "Elasticsearch")},

		// DevOps / containers
		{Name: "Docker Remote API", Title: "Docker remote API", Ports: []string{"2375,2376/tcp"}, Installed: lo.Contains(installedProfileNames, "Docker Remote API")},
		{Name: "Kubernetes API", Title: "Kubernetes API server", Ports: []string{"6443/tcp"}, Installed: lo.Contains(installedProfileNames, "Kubernetes API")},
		{Name: "Docker Swarm", Title: "Docker Swarm cluster communication", Ports: []string{"2377,7946/tcp", "7946,4789/udp"}, Installed: lo.Contains(installedProfileNames, "Docker Swarm")},

		// VPN
		{Name: "WireGuard", Title: "WireGuard VPN", Ports: []string{"51820/udp"}, Installed: lo.Contains(installedProfileNames, "WireGuard")},
		{Name: "OpenVPN", Title: "OpenVPN", Ports: []string{"1194/udp"}, Installed: lo.Contains(installedProfileNames, "OpenVPN")},

		// Email
		{Name: "SMTP", Title: "Simple Mail Transfer Protocol", Ports: []string{"25/tcp"}, Installed: lo.Contains(installedProfileNames, "SMTP")},
		{Name: "SMTPS", Title: "SMTP over SSL", Ports: []string{"465/tcp"}, Installed: lo.Contains(installedProfileNames, "SMTPS")},
		{Name: "Submission", Title: "Mail Submission Agent", Ports: []string{"587/tcp"}, Installed: lo.Contains(installedProfileNames, "Submission")},
		{Name: "IMAPS", Title: "IMAP over SSL", Ports: []string{"993/tcp"}, Installed: lo.Contains(installedProfileNames, "IMAPS")},
		{Name: "POP3S", Title: "POP3 over SSL", Ports: []string{"995/tcp"}, Installed: lo.Contains(installedProfileNames, "POP3S")},

		// DNS
		{Name: "DNS", Title: "Domain name System", Ports: []string{"53/tcp", "53/udp"}, Installed: lo.Contains(installedProfileNames, "DNS")},

		// File sharing
		{Name: "Samba", Title: "Windows file/printer sharing (Samba)", Ports: []string{"137,138/udp", "139,445/tcp"}, Installed: lo.Contains(installedProfileNames, "Samba")},
		{Name: "NFS", Title: "Network File System", Ports: []string{"111,2049/tcp", "111,2049/udp"}, Installed: lo.Contains(installedProfileNames, "NFS")},

		// Misc
		{Name: "CUPS", Title: "Common Unix Printing System", Ports: []string{"631/tcp"}, Installed: lo.Contains(installedProfileNames, "CUPS")},
		{Name: "VNC", Title: "Virtual Network Computing (remote desktop)", Ports: []string{"5900/tcp"}, Installed: lo.Contains(installedProfileNames, "VNC")},
		{Name: "Deluge", Title: "Deluge BitTorrent client", Ports: []string{"6881/tcp", "6881/udp"}, Installed: lo.Contains(installedProfileNames, "Deluge")},
		{Name: "Prometheus", Title: "Prometheus monitoring", Ports: []string{"9090/tcp"}, Installed: lo.Contains(installedProfileNames, "Prometheus")},
		{Name: "Grafana", Title: "Grafana dashboards", Ports: []string{"3000/tcp"}, Installed: lo.Contains(installedProfileNames, "Grafana")},
		{Name: "RabbitMQ", Title: "RabbitMQ message broker", Ports: []string{"5672,15672/tcp"}, Installed: lo.Contains(installedProfileNames, "RabbitMQ")},
		{Name: "Mosquitto", Title: "Mosquitto MQTT broker", Ports: []string{"1883,8883/tcp"}, Installed: lo.Contains(installedProfileNames, "Mosquitto")},
	}

	profiles = lo.Filter(profiles, func(p UFWProfile, _ int) bool {
		return !p.Installed
	})

	return profiles
}
