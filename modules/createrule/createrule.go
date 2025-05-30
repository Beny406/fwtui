package createrule

import (
	"fmt"
	"fwtui/domain/notification"
	"fwtui/utils/focusablelist"
	"fwtui/utils/oscmd"
	"fwtui/utils/result"
	stringsext "fwtui/utils/strings"
	"net"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/samber/lo"
)

// MODEL

type Field string

const (
	RuleFormPort      = "Port"
	RuleFormProtocol  = "Protocol"
	RuleFormAction    = "Action"
	RuleFormDir       = "Direction"
	RuleSourceIP      = "SourceIP"
	RuleDestinationIP = "DestinationIP"
	RuleInterface     = "Interface"
	RuleFormComment   = "Comment"
)

type RuleForm struct {
	port          string
	protocol      *focusablelist.SelectableList[Protocol]
	action        *focusablelist.SelectableList[Action]
	dir           *focusablelist.SelectableList[Direction]
	comment       string
	sourceIP      string
	destinationIP string
	interface_    *focusablelist.SelectableList[string]
	selectedField *focusablelist.SelectableList[Field]
}

func NewRuleForm() RuleForm {
	availableInterfaces, _ := GetActiveInterfaces()

	return RuleForm{
		protocol:      focusablelist.FromList(protocols),
		action:        focusablelist.FromList(actions),
		dir:           focusablelist.FromList(directions),
		interface_:    focusablelist.FromList(availableInterfaces),
		selectedField: focusablelist.FromList(fieldsForDirection(DirectionIn)),
	}
}

// UPDATE

type CreateRuleEscMsg struct{}
type CreateRuleCreatedMsg struct{}

func (f RuleForm) UpdateRuleForm(msg tea.Msg) (RuleForm, tea.Cmd) {
	form := f
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "up":
			form.selectedField.Prev()
		case "down":
			form.selectedField.Next()
		case "left":
			switch form.selectedField.Focused() {
			case RuleFormProtocol:
				form.protocol.Prev()
			case RuleFormAction:
				form.action.Prev()
			case RuleFormDir:
				form.dir.Prev()
				form.selectedField.SetItems(fieldsForDirection(form.dir.Focused()))
			case RuleInterface:
				form.interface_.Prev()
			}
			return form, nil
		case "right":
			switch form.selectedField.Focused() {
			case RuleFormProtocol:
				form.protocol.Next()
			case RuleFormAction:
				form.action.Next()
			case RuleFormDir:
				form.dir.Next()
				form.selectedField.SetItems(fieldsForDirection(form.dir.Focused()))
			case RuleInterface:
				form.interface_.Next()
			}
			return form, nil

		case "backspace":
			switch form.selectedField.Focused() {
			case RuleFormPort:
				form.port = stringsext.TrimLastChar(form.port)
			case RuleFormComment:
				form.comment = stringsext.TrimLastChar(form.comment)
			case RuleSourceIP:
				form.sourceIP = stringsext.TrimLastChar(form.sourceIP)
			case RuleDestinationIP:
				form.destinationIP = stringsext.TrimLastChar(form.destinationIP)
			}
		case "enter":
			res := f.BuildUfwCommand()
			if res.IsErr() {
				return f, notification.CreateCmd(res.Err().Error())
			}
			output := oscmd.RunCommand(res.Value())
			return f, tea.Batch(notification.CreateCmd(output), func() tea.Msg {
				return CreateRuleCreatedMsg{}
			})
		case "esc":
			return form, func() tea.Msg {
				return CreateRuleEscMsg{}
			}
		default:
			switch form.selectedField.Focused() {
			case RuleFormPort:
				form.port += key
			case RuleFormComment:
				form.comment += key
			case RuleSourceIP:
				form.sourceIP += key
			case RuleDestinationIP:
				form.destinationIP += key
			}
		}
	}
	return form, nil
}

func fieldsForDirection(dir Direction) []Field {
	baseFields := []Field{
		RuleFormPort,
		RuleFormProtocol,
		RuleFormAction,
		RuleFormDir,
		RuleFormComment,
	}

	switch dir {
	case DirectionIn:
		return append(baseFields, RuleSourceIP, RuleInterface)
	case DirectionOut:
		return append(baseFields, RuleDestinationIP)
	default:
		return baseFields // fallback in case of invalid input
	}
}

func GetActiveInterfaces() ([]string, error) {
	var result = []string{""}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		// Skip interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		result = append(result, iface.Name)
	}

	return result, nil
}

// VIEW

func (f RuleForm) ViewCreateRule() string {
	var lines []string

	for _, field := range f.selectedField.GetItems() {
		var value string
		var fieldString string

		switch field {
		case RuleFormPort:
			value = f.port
			fieldString = "Port"
		case RuleFormProtocol:
			value = string(f.protocol.Focused())
			fieldString = "Protocol"
		case RuleFormAction:
			value = string(f.action.Focused())
			fieldString = "Action"
		case RuleFormDir:
			value = string(f.dir.Focused())
			fieldString = "Direction"
		case RuleFormComment:
			value = f.comment
			fieldString = "Comment (Optional)"
		case RuleSourceIP:
			value = f.sourceIP
			fieldString = "Source IP (Optional)"
		case RuleDestinationIP:
			value = f.destinationIP
			fieldString = "Destination IP (Optional)"
		case RuleInterface:
			value = f.interface_.Focused()
			fieldString = "Interface (Optional)"
		}

		prefix := lo.Ternary(f.selectedField.Focused() == field, "> ", "  ")
		line := fmt.Sprintf("%s%s: %s", prefix, fieldString, value)
		lines = append(lines, line)
	}

	output := strings.Join(lines, "\n")
	output += "\n\n↑↓ to navigate, ←→ to change selection, type to edit, Enter to submit, Esc to cancel"
	return output
}

func (f RuleForm) BuildUfwCommand() result.Result[string] {
	// Validate port
	if strings.Contains(f.port, ":") {
		if f.protocol.Focused() == ProtocolBoth {
			return result.Err[string](fmt.Errorf("invalid protocol for port range: %s. Must be either TCP or UDP only", f.port))
		}

		split := strings.Split(f.port, ":")

		portNum1, err := strconv.Atoi(split[0])
		if err != nil || portNum1 < 1 || portNum1 > 65535 {
			return result.Err[string](fmt.Errorf("invalid port: %s", split[0]))
		}

		portNum2, err := strconv.Atoi(split[1])
		if err != nil || portNum2 < 1 || portNum2 > 65535 {
			return result.Err[string](fmt.Errorf("invalid port: %s", split[1]))
		}

		if portNum1 > portNum2 {
			return result.Err[string](fmt.Errorf("invalid port range: %s", f.port))
		}

	} else {
		portNum, err := strconv.Atoi(f.port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return result.Err[string](fmt.Errorf("invalid port: %s", f.port))
		}
	}

	// Start building the command
	parts := []string{"sudo", "ufw", string(f.action.Focused())}

	// Direction-specific parts
	switch f.dir.Focused() {
	case DirectionIn:
		if f.interface_.Focused() != "" {
			parts = append(parts, "in", "on", f.interface_.Focused())
		}

		if f.sourceIP != "" {
			if _, _, err := net.ParseCIDR(f.sourceIP); err != nil {
				if net.ParseIP(f.sourceIP) == nil {
					return result.Err[string](fmt.Errorf("invalid source IP: %s", f.sourceIP))
				}
			}
			parts = append(parts, "from", f.sourceIP)
		} else {
			parts = append(parts, "from", "any")
		}
		parts = append(parts, "to", "any")
	case DirectionOut:
		parts = append(parts, "from", "any")
		if f.destinationIP != "" {
			if _, _, err := net.ParseCIDR(f.destinationIP); err != nil {
				if net.ParseIP(f.destinationIP) == nil {
					return result.Err[string](fmt.Errorf("invalid destination IP: %s", f.destinationIP))
				}
			}
			parts = append(parts, "to", f.destinationIP)
		} else {
			parts = append(parts, "to", "any")
		}
	default:
		return result.Err[string](fmt.Errorf("invalid direction"))
	}

	// Port and protocol
	if f.protocol.Focused() == ProtocolBoth {
		parts = append(parts, "port", f.port)
	} else {
		parts = append(parts, "port", f.port, "proto", string(f.protocol.Focused()))
	}

	// Comment (optional)
	if f.comment != "" {
		sanitizedComment := strings.ReplaceAll(f.comment, `'`, `'\''`)
		parts = append(parts, "comment", fmt.Sprintf("'%s'", sanitizedComment))
	}

	return result.Ok(strings.Join(parts, " "))
}
