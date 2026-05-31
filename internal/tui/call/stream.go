package call

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/prnvbn/grpcexp/internal/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var _ Screen = &Stream{}

type streamPane int

const (
	streamPaneSend streamPane = iota
	streamPaneRecv
)

type Stream struct {
	method  protoreflect.MethodDescriptor
	builder *Builder
	client  *grpc.Client
	width   int
	height  int

	activePane  streamPane
	started     bool
	sendClosed  bool
	closed      bool
	cancel      context.CancelFunc
	requests    chan map[string]any
	events      chan grpc.StreamEvent
	transcript  []transcriptEntry
	recvCount   int
	scrollIndex int
	timestamps  bool
}

type transcriptEntry struct {
	at   time.Time
	text string
}

type streamEventMsg struct {
	event grpc.StreamEvent
}

type streamDoneMsg struct{}

func NewStream(method protoreflect.MethodDescriptor, client *grpc.Client) *Stream {
	return &Stream{
		method:  method,
		builder: NewBuilder(method.Input()),
		client:  client,
	}
}

func (f *Stream) Init() tea.Cmd {
	return nil
}

func (f *Stream) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case streamEventMsg:
		return f, f.handleStreamEvent(msg.event)
	case streamDoneMsg:
		return f, nil
	case tea.KeyMsg:
		cmd, handled := f.handleKey(msg)
		if handled {
			return f, cmd
		}
		if f.activePane == streamPaneSend {
			return f, f.builder.Update(msg)
		}
		return f, nil
	}

	if f.activePane == streamPaneSend {
		return f, f.builder.Update(msg)
	}
	return f, nil
}

func (f *Stream) View() string {
	var out strings.Builder

	out.WriteString(callHeader(f.method))
	out.WriteString("\n\n")
	out.WriteString(f.renderPanes())

	return out.String()
}

func (f *Stream) SetSize(width, height int) {
	f.width = width
	f.height = height
	if width >= 100 {
		f.builder.SetWidth((width-2)/2 - 6)
		return
	}
	f.builder.SetWidth(width - 10)
}

func (f *Stream) AcceptsTextInput() bool {
	return f.activePane == streamPaneSend && f.builder.AcceptsTextInput()
}

func (f *Stream) Cancel() {
	if f.cancel != nil {
		f.cancel()
	}
	f.closeSend()
}

func (f *Stream) handleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+y":
		if f.activePane == streamPaneRecv {
			f.copyTranscript()
		} else {
			f.copyGRPCURLCommand()
		}
		return nil, true
	case "shift+tab":
		f.togglePane()
		return nil, true
	case "ctrl+d":
		f.closeSend()
		return nil, true
	}

	if f.activePane == streamPaneRecv {
		switch msg.String() {
		case "t":
			f.timestamps = !f.timestamps
			return nil, true
		case "up":
			if f.scrollIndex > 0 {
				f.scrollIndex--
			}
			return nil, true
		case "down":
			if f.scrollIndex < f.maxScroll() {
				f.scrollIndex++
			}
			return nil, true
		}
		return nil, false
	}

	cmd, handled := f.builder.HandleKey(msg, f.sendMessage)
	return cmd, handled
}

func (f *Stream) renderPanes() string {
	send := f.renderSendPane()
	recv := f.renderReceivePane()
	if f.width < 100 {
		return send + "\n\n" + recv
	}

	gap := 2
	paneWidth := (f.width - gap) / 2
	send = lipgloss.NewStyle().Width(paneWidth).Render(send)
	recv = lipgloss.NewStyle().Width(paneWidth).Render(recv)
	return lipgloss.JoinHorizontal(lipgloss.Top, send, strings.Repeat(" ", gap), recv)
}

func (f *Stream) renderSendPane() string {
	var out strings.Builder

	title := "Send"
	if f.activePane == streamPaneSend {
		title = "> " + title
	}
	out.WriteString(headerStyle.Render(title))
	out.WriteString("\n")
	out.WriteString(f.builder.View("Send", f.activePane == streamPaneSend, f.sendClosed || f.closed))
	out.WriteString("\n\n")
	out.WriteString(labelStyle.Render("status: " + f.status()))
	out.WriteString("\n")
	out.WriteString(labelStyle.Render("tab/up/down: navigate • shift+tab: switch pane • ctrl+d: close send • ctrl+y: copy grpcurl"))

	return out.String()
}

func (f *Stream) renderReceivePane() string {
	var out strings.Builder

	title := "Receive"
	if f.activePane == streamPaneRecv {
		title = "> " + title
	}
	out.WriteString(headerStyle.Render(title))
	out.WriteString("\n")

	lines := f.visibleTranscript()
	if len(lines) == 0 {
		out.WriteString(labelStyle.Render("No stream events yet."))
		out.WriteString("\n")
	} else {
		out.WriteString(strings.Join(f.transcriptLines(lines), "\n"))
		out.WriteString("\n")
	}
	out.WriteString("\n")
	out.WriteString(labelStyle.Render("t: toggle timestamps • ctrl+y: copy transcript"))

	return out.String()
}

func (f *Stream) sendMessage() tea.Cmd {
	if f.sendClosed || f.closed {
		return nil
	}

	var cmds []tea.Cmd
	if !f.started {
		cmds = append(cmds, f.startStream())
	}

	f.requests <- f.builder.Value()
	f.appendTranscript("> sent")
	f.scrollToBottom()

	if !f.method.IsStreamingClient() {
		f.closeSend()
	}

	return tea.Batch(cmds...)
}

func (f *Stream) startStream() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.requests = make(chan map[string]any, 16)
	f.events = make(chan grpc.StreamEvent, 128)
	f.started = true

	client := f.client
	methodFullName := string(f.method.FullName())
	requests := f.requests
	events := f.events

	return tea.Batch(func() tea.Msg {
		_ = client.InvokeStreaming(ctx, methodFullName, requests, events)
		return streamDoneMsg{}
	}, f.waitForStreamEvent())
}

func (f *Stream) waitForStreamEvent() tea.Cmd {
	events := f.events
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return streamDoneMsg{}
		}
		return streamEventMsg{event: event}
	}
}

func (f *Stream) handleStreamEvent(event grpc.StreamEvent) tea.Cmd {
	switch event.Kind {
	case grpc.StreamEventResponse:
		f.recvCount++
		f.appendTranscript(fmt.Sprintf("< recv #%d\n%s", f.recvCount, strings.TrimRight(event.Message, "\n")))
		f.scrollToBottom()
		return f.waitForStreamEvent()
	case grpc.StreamEventError:
		msg := "unknown error"
		if event.Err != nil {
			msg = event.Err.Error()
		}
		f.appendTranscript("! error: " + msg)
		f.closed = true
		f.sendClosed = true
		f.scrollToBottom()
		return nil
	case grpc.StreamEventClosed:
		f.appendTranscript("x closed")
		f.closed = true
		f.sendClosed = true
		f.scrollToBottom()
		return nil
	default:
		panic(fmt.Sprintf("unknown stream event: %d", event.Kind))
	}
}

func (f *Stream) closeSend() {
	if !f.started || f.sendClosed {
		return
	}
	close(f.requests)
	f.sendClosed = true
}

func (f *Stream) togglePane() {
	if f.activePane == streamPaneSend {
		f.builder.Deactivate()
		f.activePane = streamPaneRecv
		return
	}
	f.activePane = streamPaneSend
	f.builder.Activate()
}

func (f *Stream) status() string {
	switch {
	case f.closed:
		return "closed"
	case f.sendClosed:
		return "sending closed"
	case f.started:
		return "open"
	default:
		return "idle"
	}
}

func (f *Stream) appendTranscript(text string) {
	f.transcript = append(f.transcript, transcriptEntry{
		at:   time.Now(),
		text: text,
	})
}

func (f *Stream) visibleTranscript() []transcriptEntry {
	if f.height <= 0 || len(f.transcript) == 0 {
		return f.transcript
	}

	maxLines := f.height - 8
	if maxLines < 3 {
		maxLines = 3
	}
	if maxLines >= len(f.transcript) {
		return f.transcript
	}

	start := min(f.scrollIndex, len(f.transcript)-maxLines)
	return f.transcript[start : start+maxLines]
}

func (f *Stream) transcriptLines(entries []transcriptEntry) []string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if f.timestamps {
			lines = append(lines, entry.at.Format("15:04:05.000000000")+" "+entry.text)
			continue
		}
		lines = append(lines, entry.text)
	}
	return lines
}

func (f *Stream) maxScroll() int {
	maxLines := f.height - 8
	if maxLines < 3 {
		maxLines = 3
	}
	if len(f.transcript) <= maxLines {
		return 0
	}
	return len(f.transcript) - maxLines
}

func (f *Stream) scrollToBottom() {
	f.scrollIndex = f.maxScroll()
}

func (f *Stream) copyGRPCURLCommand() {
	command, err := f.client.GRPCURLCommand(string(f.method.FullName()), f.builder.Value())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building grpcurl command: %v\n", err)
		return
	}
	if err := clipboard.WriteAll(command); err != nil {
		fmt.Fprintf(os.Stderr, "error writing to clipboard: %v\n", err)
	}
}

func (f *Stream) copyTranscript() {
	if err := clipboard.WriteAll(strings.Join(f.transcriptLines(f.transcript), "\n")); err != nil {
		fmt.Fprintf(os.Stderr, "error writing to clipboard: %v\n", err)
	}
}
